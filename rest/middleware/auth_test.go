package middleware_test

import (
	"encoding/base64"
	"fmt"
	"net/http"
	htt "net/http/httptest"
	"net/url"
	"os"
	"testing"
	"time"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/rest/middleware"
	"github.com/synapse-garden/sg-proto/rest/ws"
	"github.com/synapse-garden/sg-proto/store"
	sgt "github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
	"github.com/julienschmidt/httprouter"
	uuid "github.com/satori/go.uuid"
	xws "golang.org/x/net/websocket"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type MiddlewareSuite struct {
	db     *bolt.DB
	tmpDir string
}

var _ = Suite(&MiddlewareSuite{})

func (s *MiddlewareSuite) SetUpTest(c *C) {
	db, tmpDir, err := sgt.TempDB("sg-test")
	c.Assert(err, IsNil)
	c.Assert(db.Update(store.Wrap(
		store.Migrate(store.VerCurrent),
		store.SetupBuckets(
			auth.SessionBucket,
			auth.RefreshBucket,
			auth.ContextBucket,
		),
	)), IsNil)
	s.db, s.tmpDir = db, tmpDir
}

func (s *MiddlewareSuite) TearDownTest(c *C) {
	if db := s.db; db != nil {
		c.Assert(sgt.CleanupDB(db), IsNil)
		c.Assert(os.Remove(s.tmpDir), IsNil)
	}
}

func (s *MiddlewareSuite) TestAuthUser(c *C) {
	sess := &auth.Session{}
	c.Assert(s.db.Update(auth.NewSession(
		sess,
		time.Now().Add(1*time.Hour),
		time.Hour,
		auth.NewToken(auth.BearerType),
		auth.NewToken(auth.RefreshType),
		"friendo",
	)), IsNil)

	validToken := base64.StdEncoding.EncodeToString(sess.Token)

	r := htt.NewRequest("GET", "/foo", nil)
	r.Header.Set(string(middleware.AuthHeader), fmt.Sprintf("%s %s", auth.BearerType, validToken))
	h := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		c.Check(middleware.CtxGetUserID(r), Equals, "friendo")
		w.Write([]byte("ok"))
	}
	w := htt.NewRecorder()
	middleware.AuthUser(h, s.db, middleware.CtxSetUserID)(w, r, nil)
	c.Check(w.Body.String(), Equals, "ok")

	// TODO:
	// A request to an endpoint with the given auth scheme should
	// be rejected if header["Authorization"] is not "Bearer" and
	// present in the database.

	uu := uuid.NewV4()
	invalidToken := base64.StdEncoding.EncodeToString(uu[:])
	r = htt.NewRequest("GET", "/foo", nil)
	r.Header.Set(string(middleware.AuthHeader), fmt.Sprintf("%s %s", auth.BearerType, invalidToken))
	h = func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		c.Check(middleware.CtxGetUserID(r), Not(Equals), "friendo")
		w.Write([]byte("ok"))
	}
	w = htt.NewRecorder()
	middleware.AuthUser(h, s.db, middleware.CtxSetUserID)(w, r, nil)
	c.Check(w.Body.String(), Equals, "invalid session token\n")
}

func (s *MiddlewareSuite) TestAuthWS(c *C) {
	sess := &auth.Session{}
	c.Assert(s.db.Update(auth.NewSession(
		sess,
		time.Now().Add(1*time.Hour),
		time.Hour,
		auth.NewToken(auth.BearerType),
		auth.NewToken(auth.RefreshType),
		"friendo",
	)), IsNil)

	validToken := base64.RawURLEncoding.EncodeToString(sess.Token)
	c.Logf("valid WS token (base64.RawURLEncoding): %#q", validToken)

	h := func(w http.ResponseWriter, r *http.Request, _ httprouter.Params) {
		userID := middleware.CtxGetUserID(r)
		c.Check(userID, Equals, "friendo")
		xws.Server{
			Handshake: ws.Check,
			Handler: func(c *xws.Conn) {
				c.Write([]byte(`"hello ` + userID + `"`))
				c.Close()
			},
		}.ServeHTTP(w, r)
	}
	rt := httprouter.New()
	rt.GET("/foo", middleware.AuthWSUser(h, s.db, middleware.CtxSetUserID))
	srv := htt.NewServer(rt)

	wsURL, err := url.Parse(srv.URL + "/foo")
	c.Assert(err, IsNil)
	wsURL.Scheme = "ws"

	conn, err := xws.DialConfig(&xws.Config{
		Location: wsURL,
		Origin:   &url.URL{},
		Version:  xws.ProtocolVersionHybi13,
		// {"Sec-WebSocket-Protocol": "Bearer+12345,..."
		Protocol: []string{"Bearer+" + validToken},
	})
	c.Assert(err, IsNil)

	var got string
	c.Assert(xws.JSON.Receive(conn, &got), IsNil)
	c.Check(got, Equals, "hello friendo")
}

func (s *MiddlewareSuite) TestGetToken(c *C) {
	validToken := auth.NewToken(auth.BearerType)
	validBase64 := base64.StdEncoding.EncodeToString(validToken)

	c.Logf("valid token: %+v", validToken)

	for i, test := range []struct {
		given     string
		kind      auth.TokenType
		expect    []byte
		expectErr string
	}{{
		given:     "",
		expectErr: `no "Bearer" token provided`,
	}, {
		given:     "Basic",
		expectErr: `no "Bearer" token provided`,
	}, {
		given:     "Basic 12345 hoho",
		expectErr: `too many token fields`,
	}, {
		given:     "Basic 12345",
		expectErr: `invalid token kind "Basic", expected "Bearer"`,
	}, {
		given: "Bearer 12345",
		expectErr: "invalid token `12345`: illegal base64 " +
			"data at input byte 4",
	}, {
		given:     "Refresh " + validBase64,
		kind:      auth.RefreshType,
		expectErr: `invalid token kind "Refresh", expected "Bearer"`,
	}, {
		given:  "Bearer " + validBase64,
		expect: validToken,
	}} {
		c.Logf("test %d: %s", i, test.given)
		got, err := middleware.GetToken(auth.BearerType, test.given)
		if test.expectErr != "" {
			c.Check(err, ErrorMatches, test.expectErr)
			continue
		}
		c.Assert(err, IsNil)
		c.Check(got, DeepEquals, test.expect)
	}
}

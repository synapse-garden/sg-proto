package rest_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	htt "net/http/httptest"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream/river"
	sgt "github.com/synapse-garden/sg-proto/testing"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	htr "github.com/julienschmidt/httprouter"
	uuid "github.com/satori/go.uuid"
	ws "golang.org/x/net/websocket"
	. "gopkg.in/check.v1"
)

var _ = rest.API(new(rest.Admin))

func prepAdminAPI(c *C,
	r *htr.Router,
	api *rest.Admin,
	users ...string,
) (*htt.Server, map[string]auth.Token) {
	tokens := make(map[string]auth.Token)

	for _, user := range users {
		_, err := sgt.MakeLogin(user, "some-password", api.DB)
		c.Assert(err, IsNil)
		sesh := new(auth.Session)
		c.Assert(sgt.GetSession(user, sesh, api.DB), IsNil)
		tokens[user] = sesh.Token
	}

	c.Assert(api.Bind(r), IsNil)
	c.Assert(rest.Token{api.DB}.Bind(r), IsNil)
	c.Assert(rest.Profile{api.DB}.Bind(r), IsNil)

	// Make a testing server to run it.
	return htt.NewServer(r), tokens
}

func cleanupAdminAPI(c *C, api *rest.Admin) {
	c.Assert(api.Pub.Close(), IsNil)
	c.Assert(api.Update(func(tx *bolt.Tx) error {
		return river.DeletePub(rest.AdminNotifs, rest.NotifStream, tx)
	}), IsNil)
}

func (s *RESTSuite) TestAdminNilDB(c *C) {
	c.Assert(
		new(rest.Admin).Bind(htr.New()),
		ErrorMatches,
		"Admin DB handle must not be nil",
	)
}

func (s *RESTSuite) TestAdminNewLoginErrors(c *C) {
	var (
		tokenUUID   = uuid.NewV4()
		adminKey    = auth.Token(tokenUUID[:])
		api         = &rest.Admin{Token: adminKey, DB: s.db}
		r           = htr.New()
		srv, tokens = prepAdminAPI(c, r, api, "bob", "bodie")
	)
	defer srv.Close()

	for i, test := range []struct {
		should string

		verb, path       string
		header           http.Header
		body             interface{}
		expectStatus     int
		into, expectResp interface{}
	}{{
		should: "reject non-admin token",
		verb:   "POST", path: "/admin/logins",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp: `invalid "Admin" token provided in ` +
			`header "Authorization"` + "\n",
	}, {
		should: "reject bad admin token",
		verb:   "POST", path: "/admin/logins",
		header:       sgt.Admin(auth.Token("haha")),
		expectStatus: http.StatusUnauthorized,
		into:         new(string),
		expectResp:   "no such admin token `haha`\n",
	}, {
		should: "reject wrong HTTP method",
		verb:   "GET", path: "/admin/logins",
		header:       sgt.Admin(adminKey),
		expectStatus: http.StatusMethodNotAllowed,
		into:         new(string),
		expectResp:   "Method Not Allowed\n",
	}, {
		should: "error on bad body",
		verb:   "POST", path: "/admin/logins",
		header:       sgt.Admin(adminKey),
		body:         "hi",
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp: "failed to parse Login: " +
			"json: cannot unmarshal string into Go " +
			"value of type auth.Login\n",
	}, {
		should: "error on existing User",
		verb:   "POST", path: "/admin/logins",
		header: sgt.Admin(adminKey),
		body: &auth.Login{User: users.User{
			Name: "bob",
		}},
		expectStatus: http.StatusConflict,
		into:         new(string),
		expectResp:   "user `bob` already exists\n",
	}} {
		c.Logf("test %d: %s on %s should %s", i,
			test.verb, test.path,
			test.should,
		)
		c.Assert(sgt.ExpectResponse(r,
			test.path, test.verb, test.body,
			test.into, test.expectResp,
			test.expectStatus,
			test.header,
		), IsNil)
	}

	cleanupAdminAPI(c, api)
}

func (s *RESTSuite) TestAdminNewLoginWorks(c *C) {
	var (
		tokenUUID = uuid.NewV4()
		adminKey  = auth.Token(tokenUUID[:])
		api       = &rest.Admin{Token: adminKey, DB: s.db}
		r         = htr.New()
		srv, _    = prepAdminAPI(c, r, api)

		intoUser    = new(users.User)
		intoSession = new(auth.Session)
	)
	defer srv.Close()

	newLogin := &auth.Login{
		User:   users.User{Name: "bodo"},
		PWHash: sgt.Sha256("hello"),
	}
	c.Log("  should be able to create new Login")
	c.Assert(sgt.ExpectResponse(r,
		"/admin/logins", "POST",
		newLogin, intoUser, &(newLogin.User),
		http.StatusOK,
		sgt.Admin(adminKey),
	), IsNil)

	c.Log("  new User can create a session token")
	var rdr *bytes.Buffer
	send, err := json.Marshal(newLogin)
	c.Assert(err, IsNil)
	rdr = bytes.NewBuffer(send)
	req := htt.NewRequest("POST", "/tokens", rdr)
	w := htt.NewRecorder()
	r.ServeHTTP(w, req)
	c.Check(w.Code, Equals, http.StatusOK)

	c.Log("  after making the POST, the token should be correct")
	c.Assert(json.NewDecoder(w.Body).Decode(intoSession), IsNil)
	expectSession, err := sgt.FindSession(s.db, intoSession.Expiration)
	c.Assert(err, IsNil)
	c.Check(intoSession, DeepEquals, expectSession)

	c.Log("  new User can use his token on authed APIs normally")
	c.Assert(sgt.ExpectResponse(r,
		"/profile", "GET",
		nil, new(users.User), &(newLogin.User),
		http.StatusOK,
		sgt.Bearer(expectSession.Token),
	), IsNil)

	cleanupAdminAPI(c, api)
}

func (s *RESTSuite) TestAdminPatchProfile(c *C) {
	var (
		tokenUUID   = uuid.NewV4()
		adminKey    = auth.Token(tokenUUID[:])
		api         = &rest.Admin{Token: adminKey, DB: s.db}
		r           = htr.New()
		srv, tokens = prepAdminAPI(c, r, api, "bob", "bodie")
		notifErr    = rest.Notif{DB: s.db}.Bind(r)
	)
	defer srv.Close()
	c.Assert(notifErr, IsNil)

	// Get websocket connection for "bodie".
	connBodie, err := sgt.GetWSClient(
		base64.RawURLEncoding.EncodeToString(tokens["bodie"]),
		srv.URL+"/notifs",
	)
	c.Assert(err, IsNil)

	// Get websocket connection for "bob".
	connBob, err := sgt.GetWSClient(
		base64.RawURLEncoding.EncodeToString(tokens["bob"]),
		srv.URL+"/notifs",
	)
	c.Assert(err, IsNil)

	for i, test := range []struct {
		should string

		verb, path       string
		header           http.Header
		expectStatus     int
		into, expectResp interface{}

		expectNotifs map[*ws.Conn][]*store.ResourceBox
	}{{
		should: "reject non-admin token",
		verb:   "PATCH", path: "/admin/profiles/foo",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp: `invalid "Admin" token provided in ` +
			`header "Authorization"` + "\n",
	}, {
		should: "reject bad admin token",
		verb:   "PATCH", path: "/admin/profiles/foo",
		header:       sgt.Admin(auth.Token("haha")),
		expectStatus: http.StatusUnauthorized,
		into:         new(string),
		expectResp:   "no such admin token `haha`\n",
	}, {
		should: "reject wrong HTTP method",
		verb:   "GET", path: "/admin/profiles/foo",
		header:       sgt.Admin(adminKey),
		expectStatus: http.StatusMethodNotAllowed,
		into:         new(string),
		expectResp:   "Method Not Allowed\n",
	}, {
		should: "error on no user",
		verb:   "PATCH", path: "/admin/profiles",
		header:       sgt.Admin(adminKey),
		expectStatus: http.StatusNotFound,
		into:         new(string),
		expectResp:   "404 page not found\n",
	}, {
		should: "error on missing user",
		verb:   "PATCH", path: "/admin/profiles/bobo",
		header:       sgt.Admin(adminKey),
		expectStatus: http.StatusNotFound,
		into:         new(string),
		expectResp:   "user `bobo` not found\n",
	}, {
		should: "error on missing query params",
		verb:   "PATCH", path: "/admin/profiles/bodie",
		header:       sgt.Admin(adminKey),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp:   "no value passed for addCoin parameter\n",
	}, {
		should: "error on malformed query param",
		verb:   "PATCH", path: "/admin/profiles/bodie?addCoin=5xq",
		header:       sgt.Admin(adminKey),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp: `failed to parse coin value 5xq: ` +
			`strconv.ParseInt: parsing "5xq": ` +
			"invalid syntax\n",
	}, {
		should: "work for good query, notify Bodie",
		verb:   "PATCH", path: "/admin/profiles/bodie?addCoin=5",
		header:       sgt.Admin(adminKey),
		expectStatus: http.StatusOK,
		into:         new(users.User),
		expectResp:   &users.User{Name: "bodie", Coin: int64(5)},
		expectNotifs: map[*ws.Conn][]*store.ResourceBox{
			connBodie: {{
				Name: "users",
				Contents: map[string]interface{}{
					"name": "bodie",
					"coin": float64(5),
				},
			}},
		},
	}, {
		should: "work for good query, notify Bob",
		verb:   "PATCH", path: "/admin/profiles/bob?addCoin=50",
		header:       sgt.Admin(adminKey),
		expectStatus: http.StatusOK,
		into:         new(users.User),
		expectResp:   &users.User{Name: "bob", Coin: int64(50)},
		expectNotifs: map[*ws.Conn][]*store.ResourceBox{
			connBob: {{
				Name: "users",
				Contents: map[string]interface{}{
					"name": "bob",
					"coin": float64(50),
				},
			}},
		},
	}} {
		c.Logf("test %d: %s on %s should %s", i,
			test.verb, test.path,
			test.should,
		)
		c.Assert(sgt.ExpectResponse(r,
			test.path, test.verb, nil,
			test.into, test.expectResp,
			test.expectStatus,
			test.header,
		), IsNil)

		for conn, expects := range test.expectNotifs {
			// Check that the given user's notif conn
			// received the expected notifs.

			// ns starts as a copy of expected notifs, and
			// each is removed from 'ns' when it is found;
			// ns should be empty at the end of the loop.
			ns := expects
			var saw []*store.ResourceBox
			for _ = range expects {
				// Note this will not account for
				// unexpected notifs, but they will
				// cause problems later...
				notif := new(store.ResourceBox)
				c.Assert(ws.JSON.Receive(conn, notif), IsNil)
				saw = append(saw, notif)
			find:
				for j, n := range ns {
					// Was that one of the expected notifs?
					if reflect.DeepEqual(notif, n) {
						// If found, remove from expected
						// by re-slicing and stop search.
						ns = append(ns[:j], ns[j+1:]...)
						break find
					}
				}
			}

			c.Check(len(ns), Equals, 0)
			if c.Failed() {
				c.Logf("expected notifs: %s\n"+
					"  but never saw %s\n"+
					"  got %s\n",
					spew.Sdump(expects),
					spew.Sdump(ns),
					spew.Sdump(saw),
				)
			}
		}
	}

	c.Assert(connBodie.Close(), IsNil)
	c.Assert(connBob.Close(), IsNil)

	cleanupAdminAPI(c, api)
}

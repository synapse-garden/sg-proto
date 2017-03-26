package rest_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"io/ioutil"
	"net/http"
	htt "net/http/httptest"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/convo"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream"
	sgt "github.com/synapse-garden/sg-proto/testing"
	"github.com/synapse-garden/sg-proto/users"

	htr "github.com/julienschmidt/httprouter"
	uuid "github.com/satori/go.uuid"
	ws "golang.org/x/net/websocket"
	. "gopkg.in/check.v1"
)

func prepProfileAPI(c *C,
	r *htr.Router,
	api rest.Profile,
	users ...string,
) (*htt.Server, *rest.Convo, map[string]auth.Token) {
	tokens := make(map[string]auth.Token)

	for _, user := range users {
		_, err := sgt.MakeLogin(user, "some-password", api.DB)
		c.Assert(err, IsNil)
		sesh := new(auth.Session)
		c.Assert(sgt.GetSession(user, sesh, api.DB), IsNil)
		tokens[user] = sesh.Token
	}

	conv := &rest.Convo{DB: api.DB}

	c.Assert(rest.Token{DB: api.DB}.Bind(r), IsNil)
	c.Assert(conv.Bind(r), IsNil)
	c.Assert(rest.Incept{DB: api.DB}.Bind(r), IsNil)
	c.Assert(api.Bind(r), IsNil)

	// Make a testing server to run it.
	return htt.NewServer(r), conv, tokens

}

func (s *RESTSuite) TestProfileBind(c *C) {
	r := htr.New()
	c.Check(rest.Profile{}.Bind(r), ErrorMatches, ".*not be nil")
	c.Check(rest.Profile{DB: s.db}.Bind(r), IsNil)
}

func (s *RESTSuite) TestProfileGet(c *C) {
	var (
		api               = rest.Profile{DB: s.db}
		r                 = htr.New()
		srv, conv, tokens = prepProfileAPI(c, r, api, "bob", "bodie")
	)
	defer srv.Close()
	defer cleanupConvoAPI(c, *conv)

	uu := uuid.NewV4()
	someToken := auth.Token(uu[:])

	for i, test := range []struct {
		should string

		path             string
		header           http.Header
		expectStatus     int
		into, expectResp interface{}
		expectHeaders    []http.Header
	}{{
		should:       "not allow GET with unauthorized token",
		path:         "/profile",
		header:       sgt.Bearer(someToken),
		expectStatus: http.StatusUnauthorized,
		into:         new(string),
		expectResp:   "invalid session token\n",
	}, {
		should:       "map GET to authorized token's user",
		path:         "/profile",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusOK,
		into:         new(users.User),
		expectResp:   &users.User{Name: "bodie"},
	}, {
		should:       "map GET to authorized token's user",
		path:         "/profile",
		header:       sgt.Bearer(tokens["bob"]),
		expectStatus: http.StatusOK,
		into:         new(users.User),
		expectResp:   &users.User{Name: "bob"},
	}} {
		c.Logf("test %d: %s on %s should %s", i,
			"GET", test.path,
			test.should,
		)
		c.Assert(sgt.ExpectResponse(r,
			test.path, "GET", nil,
			test.into, test.expectResp,
			test.expectStatus,
			test.header,
			test.expectHeaders...,
		), IsNil)
	}
}

func (s *RESTSuite) TestProfileDelete(c *C) {
	// This test is supposed to show that after a user's profile is
	// deleted, that user's tokens are removed, and login disabled.
	var (
		api               = rest.Profile{DB: s.db}
		r                 = htr.New()
		srv, conv, tokens = prepProfileAPI(c, r, api, "bob", "bodie")
	)
	defer srv.Close()
	defer cleanupConvoAPI(c, *conv)

	uu := uuid.NewV4()
	someToken := auth.Token(uu[:])

	// Create a ticket to incept later.
	tick := incept.Ticket(uuid.NewV4())
	c.Assert(s.db.Update(incept.NewTickets(tick)), IsNil)

	for i, test := range []struct {
		should string

		verb, path             string
		header                 http.Header
		expectStatus           int
		body, into, expectResp interface{}
		expectHeaders          []http.Header
	}{{
		should: "GET account by token before disable OK",
		verb:   "GET", path: "/profile",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusOK,
		into:         new(users.User),
		expectResp:   &users.User{Name: "bodie"},
	}, {
		should: "reject unauthorized DELETE on /profile",
		verb:   "DELETE", path: "/profile",
		header:       sgt.Bearer(someToken),
		expectStatus: http.StatusUnauthorized,
		into:         new(string),
		expectResp:   "invalid session token\n",
	}, {
		should: "map token to account for DELETE",
		verb:   "DELETE", path: "/profile",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusOK,
		into:         new(string),
		expectResp:   "",
	}, {
		should: "fail to auth now; bodie's token is gone",
		verb:   "GET", path: "/profile",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusUnauthorized,
		into:         new(string),
		expectResp:   "invalid session token\n",
	}, {
		should: "fail to login now, login disabled",
		verb:   "POST", path: "/tokens",
		body: auth.Login{
			User:   users.User{Name: "bodie"},
			PWHash: sgt.Sha256("some-password"),
		},
		expectStatus: http.StatusUnauthorized,
		into:         new(string),
		expectResp:   "login for user `bodie` disabled\n",
	}, {
		should: "not permit user creation with DELETEd name",
		verb:   "POST", path: "/incept/" + tick.String(),
		expectStatus: http.StatusConflict,
		body: auth.Login{
			User:   users.User{Name: "bodie"},
			PWHash: sgt.Sha256("dadada"),
		},
		into:       new(string),
		expectResp: "login for user `bodie` already exists\n",
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
			test.expectHeaders...,
		), IsNil)
	}
}

func (s *RESTSuite) TestProfileDeleteHangups(c *C) {
	// This test is supposed to create a convo, connect to it, and
	// show that when the profile is deleted, the user is hung up.
	var (
		api               = rest.Profile{DB: s.db}
		r                 = htr.New()
		srv, conv, tokens = prepProfileAPI(c, r, api, "bob", "bodie")
	)
	defer srv.Close()
	defer cleanupConvoAPI(c, *conv)

	notifs := rest.Notif{DB: s.db}
	c.Assert(notifs.Bind(r), IsNil)

	toPOST := &convo.Convo{Group: users.Group{
		Owner:   "bodie",
		Readers: map[string]bool{"bodie": true, "bob": true},
		Writers: map[string]bool{"bodie": true, "bob": true},
	}}

	// POST the new Convo to Create it.
	send, err := json.Marshal(toPOST)
	c.Assert(err, IsNil)

	req := htt.NewRequest("POST", "/convos", bytes.NewBuffer(send))
	req.Header = sgt.Bearer(tokens["bodie"])
	w := htt.NewRecorder()
	r.ServeHTTP(w, req)
	c.Check(w.Code, Equals, http.StatusOK)

	got := new(convo.Convo)
	c.Assert(json.Unmarshal(w.Body.Bytes(), got), IsNil)
	c.Check(got.Owner, Equals, toPOST.Owner)
	c.Check(got.Writers, DeepEquals, toPOST.Writers)
	c.Check(got.Readers, DeepEquals, toPOST.Readers)

	// Get two websocket connections.
	conn1, err := sgt.GetWSClient(
		base64.RawURLEncoding.EncodeToString(tokens["bodie"]),
		srv.URL+"/convos/"+got.ID+"/start",
	)
	c.Assert(err, IsNil)
	conn2, err := sgt.GetWSClient(
		base64.RawURLEncoding.EncodeToString(tokens["bob"]),
		srv.URL+"/convos/"+got.ID+"/start",
	)
	c.Assert(err, IsNil)

	// Make sure sending and receiving works for both.
	c.Assert(ws.JSON.Send(conn1, stream.Message{Content: "hello"}), IsNil)
	msgGot := new(convo.Message)
	c.Assert(ws.JSON.Receive(conn2, msgGot), IsNil)
	c.Check(msgGot.Sender, Equals, "bodie")

	c.Assert(ws.JSON.Send(conn2, stream.Message{Content: "hello"}), IsNil)
	msgGot = new(convo.Message)
	c.Assert(ws.JSON.Receive(conn1, msgGot), IsNil)
	c.Check(msgGot.Sender, Equals, "bob")

	// Get a notifs connection to see bodie be removed.
	wsToken := base64.RawURLEncoding.EncodeToString(tokens["bob"])
	bobNotif, err := sgt.GetWSClient(wsToken, srv.URL+"/notifs")
	c.Assert(err, IsNil)

	c.Assert(sgt.ExpectResponse(r,
		"/profile", "DELETE", nil,
		new(string), "",
		http.StatusOK,
		sgt.Bearer(tokens["bodie"]),
	), IsNil)

	// conn1 should have been closed; bob's notifs should receive a
	// notif that bodie left.
	bs, err := ioutil.ReadAll(conn1)
	c.Check(string(bs), Equals, "stream hung up: EOF")

	intoNotif := new(store.ResourceBox)
	c.Assert(ws.JSON.Receive(bobNotif, intoNotif), IsNil)
	c.Check(intoNotif, DeepEquals, &store.ResourceBox{
		Name: "convo-disconnected",
		Contents: map[string]interface{}{
			"userID":  "bodie",
			"convoID": got.ID,
		}},
	)

	c.Assert(conn2.Close(), IsNil)
	c.Assert(bobNotif.Close(), IsNil)
}

func (s *RESTSuite) TestProfileOptions(c *C) {
	var (
		api = rest.Profile{DB: s.db}
		r   = htr.New()
		err = api.Bind(r)
		srv = htt.NewServer(r)
	)

	defer srv.Close()

	c.Assert(err, IsNil)

	c.Assert(sgt.ExpectResponse(r,
		"/profile", "OPTIONS", nil,
		new(string), "",
		http.StatusOK,
		nil,
		sgt.Options("GET", "DELETE", "OPTIONS"),
	), IsNil)
}

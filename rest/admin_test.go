package rest_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	htt "net/http/httptest"
	"reflect"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/convo"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream"
	sgt "github.com/synapse-garden/sg-proto/testing"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/davecgh/go-spew/spew"
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
) (*htt.Server, rest.Cleanup, map[string]auth.Token) {
	tokens := make(map[string]auth.Token)

	for _, user := range users {
		_, err := sgt.MakeLogin(user, "some-password", api.DB)
		c.Assert(err, IsNil)
		sesh := new(auth.Session)
		c.Assert(sgt.GetSession(user, sesh, api.DB), IsNil)
		tokens[user] = sesh.Token
	}

	cc, err := api.Bind(r)
	defer func() {
		if c.Failed() {
			cc()
		}
	}()
	c.Assert(err, IsNil)
	_, err = rest.Token{api.DB}.Bind(r)
	c.Assert(err, IsNil)
	_, err = rest.Profile{api.DB}.Bind(r)
	c.Assert(err, IsNil)

	// Make a testing server to run it.
	return htt.NewServer(r), cc, tokens
}

func (s *RESTSuite) TestAdminNilDB(c *C) {
	_, err := new(rest.Admin).Bind(nil)
	c.Assert(err, ErrorMatches, "nil Admin DB handle")
}

func (s *RESTSuite) TestAdminGetAllProfiles(c *C) {
	var (
		tokenUUID  = uuid.NewV4()
		adminKey   = auth.Token(tokenUUID[:])
		api        = &rest.Admin{Token: adminKey, DB: s.db}
		r          = htr.New()
		srv, cc, _ = prepAdminAPI(c, r, api, "bob", "bodie")
	)
	defer srv.Close()
	defer cc()

	uu := uuid.NewV4()
	tok := auth.Token(uu[:])

	for i, test := range []struct {
		should string

		path             string
		header           http.Header
		expectStatus     int
		into, expectResp interface{}
		expectHeaders    []http.Header
	}{{
		should:       "reject bad admin token",
		path:         "/admin/profiles",
		header:       sgt.Admin(tok),
		expectStatus: http.StatusUnauthorized,
		into:         new(string),
		expectResp:   "no such admin token `" + tok.String() + "`\n",
	}, {
		should:       "get all User profiles with good token",
		path:         "/admin/profiles",
		header:       sgt.Admin(adminKey),
		expectStatus: http.StatusOK,
		into:         new([]users.User),
		expectResp:   &([]users.User{{Name: "bob"}, {Name: "bodie"}}),
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

func (s *RESTSuite) TestAdminNewLoginErrors(c *C) {
	var (
		tokenUUID       = uuid.NewV4()
		adminKey        = auth.Token(tokenUUID[:])
		api             = &rest.Admin{Token: adminKey, DB: s.db}
		r               = htr.New()
		srv, cc, tokens = prepAdminAPI(c, r, api, "bob", "bodie")
	)
	defer srv.Close()
	defer cc()

	uu := uuid.NewV4()
	tok := auth.Token(uu[:])

	for i, test := range []struct {
		should string

		verb, path       string
		header           http.Header
		body             interface{}
		expectStatus     int
		into, expectResp interface{}
		expectHeaders    []http.Header
	}{{
		should: "reject non-admin token",
		verb:   "POST", path: "/admin/logins",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp: `invalid token kind "Bearer", expected ` +
			`"Admin"` + "\n",
	}, {
		should: "reject bad admin token",
		verb:   "POST", path: "/admin/logins",
		header:       sgt.Admin(tok),
		expectStatus: http.StatusUnauthorized,
		into:         new(string),
		expectResp:   "no such admin token `" + tok.String() + "`\n",
	}, {
		should: "reject wrong HTTP method",
		verb:   "GET", path: "/admin/logins",
		header:       sgt.Admin(adminKey),
		expectStatus: http.StatusMethodNotAllowed,
		into:         new(string),
		expectResp:   "Method Not Allowed\n",
		expectHeaders: []http.Header{
			sgt.FailHeader,
			sgt.Options("POST", "OPTIONS"),
		},
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
			test.expectHeaders...,
		), IsNil)
	}
}

func (s *RESTSuite) TestAdminNewLoginWorks(c *C) {
	var (
		tokenUUID  = uuid.NewV4()
		adminKey   = auth.Token(tokenUUID[:])
		api        = &rest.Admin{Token: adminKey, DB: s.db}
		r          = htr.New()
		srv, cc, _ = prepAdminAPI(c, r, api)

		intoUser    = new(users.User)
		intoSession = new(auth.Session)
	)
	defer srv.Close()
	defer cc()

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
}

func (s *RESTSuite) TestAdminPatchProfile(c *C) {
	var (
		tokenUUID       = uuid.NewV4()
		adminKey        = auth.Token(tokenUUID[:])
		api             = &rest.Admin{Token: adminKey, DB: s.db}
		r               = htr.New()
		srv, cc, tokens = prepAdminAPI(c, r, api, "bob", "bodie")
		_, notifErr     = rest.Notif{DB: s.db}.Bind(r)
	)
	defer srv.Close()
	defer cc()

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

	defer func() {
		c.Assert(connBodie.Close(), IsNil)
		c.Assert(connBob.Close(), IsNil)
	}()

	uu := uuid.NewV4()
	tok := auth.Token(uu[:])

	for i, test := range []struct {
		should string

		verb, path       string
		header           http.Header
		expectStatus     int
		into, expectResp interface{}

		expectNotifs  map[*ws.Conn][]*store.ResourceBox
		expectHeaders []http.Header
	}{{
		should: "reject non-admin token",
		verb:   "PATCH", path: "/admin/profiles/foo",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp: `invalid token kind "Bearer", expected ` +
			`"Admin"` + "\n",
	}, {
		should: "reject bad admin token",
		verb:   "PATCH", path: "/admin/profiles/foo",
		header:       sgt.Admin(tok),
		expectStatus: http.StatusUnauthorized,
		into:         new(string),
		expectResp:   "no such admin token `" + tok.String() + "`\n",
	}, {
		should: "reject wrong HTTP method",
		verb:   "GET", path: "/admin/profiles/foo",
		header:       sgt.Admin(adminKey),
		expectStatus: http.StatusMethodNotAllowed,
		into:         new(string),
		expectResp:   "Method Not Allowed\n",
		expectHeaders: []http.Header{
			sgt.FailHeader,
			sgt.Options("PATCH", "OPTIONS"),
		},
	}, {
		should: "error on no user",
		verb:   "PATCH", path: "/admin/profiles/",
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
			test.expectHeaders...,
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
}

func (s *RESTSuite) TestAdminGetTickets(c *C) {
	var (
		tokenUUID = uuid.NewV4()
		adminKey  = auth.Token(tokenUUID[:])
		api       = &rest.Admin{Token: adminKey, DB: s.db}
		r         = htr.New()
		cc, err   = api.Bind(r)
	)

	c.Assert(err, IsNil)
	defer cc()

	c.Log("GET /admin/tickets returns empty array")

	c.Assert(sgt.ExpectResponse(r,
		"/admin/tickets", "GET", nil,
		new([]incept.Ticket), new([]incept.Ticket),
		200,
		sgt.Admin(adminKey),
	), IsNil)

	var (
		t1     = incept.Ticket(uuid.NewV4())
		t2     = incept.Ticket(uuid.NewV4())
		expect = []incept.Ticket{t1, t2}
	)
	if t2.String() < t1.String() {
		expect = []incept.Ticket{t2, t1}
	}

	c.Assert(s.db.Update(incept.NewTickets(t1, t2)), IsNil)

	for i, test := range []struct {
		should string

		verb, path       string
		header           http.Header
		expectStatus     int
		into, expectResp interface{}

		expectNotifs  map[*ws.Conn][]*store.ResourceBox
		expectHeaders []http.Header
	}{{
		should: "reject wrong HTTP method",
		verb:   "DELETE", path: "/admin/tickets",
		header:       sgt.Admin(adminKey),
		expectStatus: http.StatusMethodNotAllowed,
		into:         new(string),
		expectResp:   "Method Not Allowed\n",
		expectHeaders: []http.Header{
			sgt.FailHeader,
			sgt.Options("GET", "POST", "OPTIONS"),
		},
	}, {
		should: "return expected tickets",
		verb:   "GET", path: "/admin/tickets",
		header:       sgt.Admin(adminKey),
		expectStatus: http.StatusOK,
		into:         new([]incept.Ticket),
		expectResp:   &expect,
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
			test.expectHeaders...,
		), IsNil)
	}
}

func (s *RESTSuite) TestAdminDeleteUser(c *C) {
	var (
		tokenUUID       = uuid.NewV4()
		adminKey        = auth.Token(tokenUUID[:])
		api             = &rest.Admin{Token: adminKey, DB: s.db}
		r               = htr.New()
		srv, cc, tokens = prepAdminAPI(c, r, api, "bob", "bodie")
		convoAPI        = &rest.Convo{DB: s.db}
		cc2, convoErr   = convoAPI.Bind(r)
		_, notifErr     = rest.Notif{DB: s.db}.Bind(r)
	)
	defer srv.Close()
	defer cc()
	defer cc2()

	c.Assert(convoErr, IsNil)
	c.Assert(notifErr, IsNil)

	// Create a convo to connect to.
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

	// Make sure it worked.
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
	c.Assert(ws.JSON.Receive(conn1, msgGot), IsNil)
	c.Check(msgGot.Sender, Equals, "bob")

	// Get a notifs connection to see bodie be removed.
	wsToken := base64.RawURLEncoding.EncodeToString(tokens["bob"])
	bobNotif, err := sgt.GetWSClient(wsToken, srv.URL+"/notifs")
	c.Assert(err, IsNil)

	// Bad admin key should be rejected.
	uu := uuid.NewV4()
	someToken := auth.Token(uu[:])

	c.Assert(sgt.ExpectResponse(r,
		"/admin/users/bodie", "DELETE", nil,
		new(string),
		"no such admin token `"+someToken.String()+"`\n",
		http.StatusUnauthorized,
		sgt.Admin(someToken),
	), IsNil)

	// Convos still work, user still exists.
	c.Assert(ws.JSON.Send(conn1, stream.Message{Content: "hello"}), IsNil)
	c.Assert(ws.JSON.Receive(conn2, msgGot), IsNil)
	c.Check(msgGot.Sender, Equals, "bodie")

	c.Assert(ws.JSON.Send(conn2, stream.Message{Content: "hello"}), IsNil)
	c.Assert(ws.JSON.Receive(conn1, msgGot), IsNil)
	c.Check(msgGot.Sender, Equals, "bob")

	// Good admin key permits DELETE of user.
	c.Assert(sgt.ExpectResponse(r,
		"/admin/users/bodie", "DELETE", nil,
		new(string), "",
		http.StatusOK,
		sgt.Admin(adminKey),
	), IsNil)

	// bob sees a hangup notif for bodie.
	intoNotif := new(store.ResourceBox)
	c.Assert(ws.JSON.Receive(bobNotif, intoNotif), IsNil)
	c.Check(intoNotif, DeepEquals, &store.ResourceBox{
		Name: "convo-disconnected",
		Contents: map[string]interface{}{
			"userID":  "bodie",
			"convoID": got.ID,
		}},
	)

	// Create a valid ticket for incept attempt.
	tick := incept.Ticket(uuid.NewV4())
	c.Assert(s.db.Update(incept.NewTickets(tick)), IsNil)

	// Bind the Incept API for testing.
	_, err = rest.Incept{DB: api.DB}.Bind(r)
	c.Assert(err, IsNil)

	// bodie's login is disabled; new bodie user cannot be created;
	// bodie's sessions are cleared.
	for i, test := range []struct {
		should string

		verb, path             string
		header                 http.Header
		expectStatus           int
		body, into, expectResp interface{}
		expectHeaders          []http.Header
	}{{
		should: "fail to auth now; bodie's token is gone",
		verb:   "GET", path: "/profile",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusUnauthorized,
		into:         new(string),
		expectResp:   "invalid session token\n",
	}, {
		should: "fail to POST to /tokens now, login disabled",
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

	c.Assert(conn1.Close(), IsNil)
	c.Assert(conn2.Close(), IsNil)
	c.Assert(bobNotif.Close(), IsNil)
}

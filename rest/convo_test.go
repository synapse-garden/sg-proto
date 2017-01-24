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
	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/stream"
	"github.com/synapse-garden/sg-proto/stream/river"
	sgt "github.com/synapse-garden/sg-proto/testing"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	"github.com/julienschmidt/httprouter"
	uuid "github.com/satori/go.uuid"
	ws "golang.org/x/net/websocket"
	. "gopkg.in/check.v1"
)

func prepConvoAPI(c *C,
	r *httprouter.Router,
	api *rest.Convo,
	names ...string,
) (*htt.Server, map[string]auth.Token) {
	tokens := make(map[string]auth.Token)

	for _, user := range names {
		_, err := sgt.MakeLogin(user, "some-password", api.DB)
		c.Assert(err, IsNil)
		sesh := new(auth.Session)
		c.Assert(sgt.GetSession(user, sesh, api.DB), IsNil)
		tokens[user] = sesh.Token
	}

	c.Assert(api.Bind(r), IsNil)

	// Make a testing server to run it.
	return htt.NewServer(r), tokens
}

func cleanupConvoAPI(c *C, api rest.Convo) {
	c.Assert(api.Pub.Close(), IsNil)
	c.Assert(api.Update(func(tx *bolt.Tx) error {
		return river.DeletePub(rest.ConvoNotifs, rest.NotifStream, tx)
	}), IsNil)
}

func (s *RESTSuite) TestConvoCreate(c *C) {
	r := httprouter.New()
	api := rest.Convo{DB: s.db}
	srv, tokens := prepConvoAPI(c, r, &api, "bodie", "bob", "jim")
	defer srv.Close()

	conv := &convo.Convo{Group: users.Group{
		Owner:   "bodie",
		Readers: map[string]bool{"bodie": true, "bob": true},
		Writers: map[string]bool{"bodie": true, "bob": true},
	}}

	// POST the new Convo to Create it.
	send, err := json.Marshal(conv)
	c.Assert(err, IsNil)

	req := htt.NewRequest("POST", "/convos", bytes.NewBuffer(send))
	req.Header = sgt.Bearer(tokens["bodie"])
	w := htt.NewRecorder()
	r.ServeHTTP(w, req)
	c.Check(w.Code, Equals, http.StatusOK)
	got := new(convo.Convo)
	c.Assert(json.Unmarshal(w.Body.Bytes(), got), IsNil)
	c.Check(got.Owner, Equals, conv.Owner)
	c.Check(got.Writers, DeepEquals, conv.Writers)
	c.Check(got.Readers, DeepEquals, conv.Readers)

	// After creation, GET on /convo/:id/messages should return [].
	c.Assert(sgt.ExpectResponse(r,
		"/convos/"+got.ID+"/messages", "GET",
		nil, new([]convo.Message), &[]convo.Message{},
		http.StatusOK,
		sgt.Bearer(tokens["bodie"]),
	), IsNil)

	// After creation, GET on /convo/:id should return the convo.
	c.Assert(sgt.ExpectResponse(r,
		"/convos/"+got.ID, "GET",
		nil, new(convo.Convo), got,
		http.StatusOK,
		sgt.Bearer(tokens["bodie"]),
	), IsNil)

	// An unauthorized user cannot GET the convo.
	c.Assert(sgt.ExpectResponse(r,
		"/convos/"+got.ID, "GET",
		nil, "", "user `jim` not a member of convo `"+got.ID+"`\n",
		http.StatusUnauthorized,
		sgt.Bearer(tokens["jim"]),
	), IsNil)

	cleanupConvoAPI(c, api)
}

func (s *RESTSuite) TestConvoPut(c *C) {
	r := httprouter.New()
	api := rest.Convo{DB: s.db}
	srv, tokens := prepConvoAPI(c, r, &api, "bodie", "bob", "jim")
	defer srv.Close()

	conv := &convo.Convo{Group: users.Group{
		Owner:   "bodie",
		Readers: map[string]bool{"bodie": true, "bob": true},
		Writers: map[string]bool{"bodie": true, "bob": true},
	}}

	c.Log("POST the new Convo to Create it.")
	send, err := json.Marshal(conv)
	c.Assert(err, IsNil)

	req := htt.NewRequest("POST", "/convos", bytes.NewBuffer(send))
	req.Header = sgt.Bearer(tokens["bodie"])
	w := htt.NewRecorder()
	r.ServeHTTP(w, req)
	c.Check(w.Code, Equals, http.StatusOK)

	got := new(convo.Convo)
	c.Assert(json.Unmarshal(w.Body.Bytes(), got), IsNil)
	c.Check(got.Owner, Equals, conv.Owner)
	c.Check(got.Writers, DeepEquals, conv.Writers)
	c.Check(got.Readers, DeepEquals, conv.Readers)

	newConv := *got
	newConv.Writers = map[string]bool{
		"bodie": true,
		"bob":   true,
		"jim":   true,
	}

	c.Log("An unauthorized user cannot PUT the convo.")
	c.Assert(sgt.ExpectResponse(r,
		"/convos/"+got.ID, "PUT",
		newConv, "", "user `bob` does not own Convo `"+got.ID+"`\n",
		http.StatusUnauthorized,
		sgt.Bearer(tokens["bob"]),
	), IsNil)

	c.Log("An authorized user can PUT the convo.")
	into := new(convo.Convo)
	c.Assert(sgt.ExpectResponse(r,
		"/convos/"+got.ID, "PUT",
		newConv, into, &newConv,
		http.StatusOK,
		sgt.Bearer(tokens["bodie"]),
	), IsNil)

	c.Log("A nonexistent user cannot be added to a Convo using a PUT.")
	newConv.Readers = map[string]bool{"bodie": true, "jimbles": true}
	c.Assert(sgt.ExpectResponse(r,
		"/convos/"+got.ID, "PUT",
		newConv, "", "invalid Convo: user `jimbles` not found\n",
		http.StatusNotFound,
		sgt.Bearer(tokens["bodie"]),
	), IsNil)

	c.Log("A nonexistent Convo cannot be PUT.")
	newConv.Readers = map[string]bool{"bodie": true, "bob": true}
	someID := uuid.NewV4().String()
	c.Assert(sgt.ExpectResponse(r,
		"/convos/"+someID, "PUT",
		newConv, "", "invalid Convo: no such convo `"+someID+"`\n",
		http.StatusNotFound,
		sgt.Bearer(tokens["bodie"]),
	), IsNil)

	connBodie, err := sgt.GetWSClient(
		base64.RawURLEncoding.EncodeToString(tokens["bodie"]),
		srv.URL+"/convos/"+got.ID+"/start",
	)
	c.Assert(err, IsNil)
	connBob, err := sgt.GetWSClient(
		base64.RawURLEncoding.EncodeToString(tokens["bob"]),
		srv.URL+"/convos/"+got.ID+"/start",
	)
	c.Assert(err, IsNil)
	connJim, err := sgt.GetWSClient(
		base64.RawURLEncoding.EncodeToString(tokens["jim"]),
		srv.URL+"/convos/"+got.ID+"/start",
	)
	c.Assert(err, IsNil)

	c.Log("If someone updates the convo so Bob is removed, his " +
		"websocket will be closed.")
	newConv.Readers = map[string]bool{"bodie": true, "jim": true}
	newConv.Writers = map[string]bool{"bodie": true, "jim": true}
	into = new(convo.Convo)
	c.Assert(sgt.ExpectResponse(r,
		"/convos/"+got.ID, "PUT",
		newConv, into, &newConv,
		http.StatusOK,
		sgt.Bearer(tokens["bodie"]),
	), IsNil)

	c.Log("Bob's Bus river is hung up now.")
	bs, err := ioutil.ReadAll(connBob)
	c.Assert(err, IsNil) // ReadAll EOF => nil error
	c.Check(string(bs), Equals, "stream hung up: EOF")

	c.Log("Jim and Bodie can still use the Bus river.")
	c.Assert(ws.JSON.Send(connJim, stream.Message{Content: "hello"}), IsNil)
	msgGot := new(convo.Message)
	c.Assert(ws.JSON.Receive(connBodie, msgGot), IsNil)
	c.Check(msgGot.Sender, Equals, "jim")

	c.Assert(ws.JSON.Send(connBodie, stream.Message{Content: "hello"}), IsNil)
	msgGot = new(convo.Message)
	c.Assert(ws.JSON.Receive(connJim, msgGot), IsNil)
	c.Check(msgGot.Sender, Equals, "bodie")

	c.Assert(connBodie.Close(), IsNil)
	c.Assert(connJim.Close(), IsNil)

	cleanupConvoAPI(c, api)
}

func (s *RESTSuite) TestConvoDelete(c *C) {
	r := httprouter.New()
	api := rest.Convo{DB: s.db}
	srv, tokens := prepConvoAPI(c, r, &api, "bodie", "bob", "jim")
	defer srv.Close()

	conv := &convo.Convo{Group: users.Group{
		Owner:   "bodie",
		Readers: map[string]bool{"bodie": true, "bob": true},
		Writers: map[string]bool{"bodie": true, "bob": true},
	}}

	// POST the new Convo to Create it.
	send, err := json.Marshal(conv)
	c.Assert(err, IsNil)

	req := htt.NewRequest("POST", "/convos", bytes.NewBuffer(send))
	req.Header = sgt.Bearer(tokens["bodie"])
	w := htt.NewRecorder()
	r.ServeHTTP(w, req)
	c.Check(w.Code, Equals, http.StatusOK)

	got := new(convo.Convo)
	c.Assert(json.Unmarshal(w.Body.Bytes(), got), IsNil)
	c.Check(got.Owner, Equals, conv.Owner)
	c.Check(got.Writers, DeepEquals, conv.Writers)
	c.Check(got.Readers, DeepEquals, conv.Readers)

	// An unauthorized user cannot DELETE the convo.
	c.Assert(sgt.ExpectResponse(r,
		"/convos/"+got.ID, "DELETE",
		nil, "", "convo `"+got.ID+"` not owned by user `jim`\n",
		http.StatusUnauthorized,
		sgt.Bearer(tokens["jim"]),
	), IsNil)

	// A non-owner member of the convo cannot DELETE the convo.
	c.Assert(sgt.ExpectResponse(r,
		"/convos/"+got.ID, "DELETE",
		nil, "", "convo `"+got.ID+"` not owned by user `bob`\n",
		http.StatusUnauthorized,
		sgt.Bearer(tokens["bob"]),
	), IsNil)

	// The convo's owner can DELETE the convo.
	c.Assert(sgt.ExpectResponse(r,
		"/convos/"+got.ID, "DELETE",
		nil, "", "",
		http.StatusOK,
		sgt.Bearer(tokens["bodie"]),
	), IsNil)

	// After the convo is DELETEd, it is not found (404.)
	c.Assert(sgt.ExpectResponse(r,
		"/convos/"+got.ID, "GET",
		nil, "", "no such convo `"+got.ID+"`\n",
		http.StatusNotFound,
		sgt.Bearer(tokens["bodie"]),
	), IsNil)

	// After the convo is DELETEd, its messages are not found (404.)
	c.Assert(sgt.ExpectResponse(r,
		"/convos/"+got.ID+"/messages", "GET",
		nil, "", "no such convo `"+got.ID+"`\n",
		http.StatusNotFound,
		sgt.Bearer(tokens["bodie"]),
	), IsNil)

	cleanupConvoAPI(c, api)
}

// GET messages
// POST edits to messages by ID
// messages saved by stream /messages/:stream_id/ prefixed by user ID in
//  chrono order?
// user has chats
//

func (s *RESTSuite) TestConvoHangupWhileDelete(c *C) {
	// This test identifies a 500 occasionally returned on convo
	// delete when the websocket is also being hung up.
	r := httprouter.New()
	api := rest.Convo{DB: s.db}
	srv, tokens := prepConvoAPI(c, r, &api, "bodie")
	defer srv.Close()

	conv := &convo.Convo{Group: users.Group{
		Owner:   "bodie",
		Readers: map[string]bool{"bodie": true},
		Writers: map[string]bool{"bodie": true},
	}}

	// POST the new Convo to Create it.
	send, err := json.Marshal(conv)
	c.Assert(err, IsNil)

	req := htt.NewRequest("POST", "/convos", bytes.NewBuffer(send))
	req.Header = sgt.Bearer(tokens["bodie"])
	w := htt.NewRecorder()
	r.ServeHTTP(w, req)
	c.Check(w.Code, Equals, http.StatusOK)

	got := new(convo.Convo)
	c.Assert(json.Unmarshal(w.Body.Bytes(), got), IsNil)
	c.Check(got.Owner, Equals, conv.Owner)
	c.Check(got.Writers, DeepEquals, conv.Writers)
	c.Check(got.Readers, DeepEquals, conv.Readers)

	// Get two websocket connections.
	conn1, err := sgt.GetWSClient(
		base64.RawURLEncoding.EncodeToString(tokens["bodie"]),
		srv.URL+"/convos/"+got.ID+"/start",
	)
	c.Assert(err, IsNil)
	conn2, err := sgt.GetWSClient(
		base64.RawURLEncoding.EncodeToString(tokens["bodie"]),
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
	c.Check(msgGot.Sender, Equals, "bodie")

	// Deleting the Convo while a websocket is closing is not a
	// problem.
	c.Assert(conn2.Close(), IsNil)
	c.Assert(sgt.ExpectResponse(r,
		"/convos/"+got.ID, "DELETE",
		nil, "", "",
		http.StatusOK,
		sgt.Bearer(tokens["bodie"]),
	), IsNil)

	// Closing the other one is fine too.
	c.Assert(conn1.Close(), IsNil)

	cleanupConvoAPI(c, api)
}

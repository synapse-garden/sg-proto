package rest_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	htt "net/http/httptest"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/convo"
	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/stream/river"
	sgt "github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
	"github.com/julienschmidt/httprouter"
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

	conv := &convo.Convo{
		Owner:   "bodie",
		Readers: map[string]bool{"bodie": true, "bob": true},
		Writers: map[string]bool{"bodie": true, "bob": true},
	}

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

func (s *RESTSuite) TestConvoDelete(c *C) {
	r := httprouter.New()
	api := rest.Convo{DB: s.db}
	srv, tokens := prepConvoAPI(c, r, &api, "bodie", "bob", "jim")
	defer srv.Close()

	conv := &convo.Convo{
		Owner:   "bodie",
		Readers: map[string]bool{"bodie": true, "bob": true},
		Writers: map[string]bool{"bodie": true, "bob": true},
	}

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

package rest_test

import (
	"encoding/base64"
	"io/ioutil"
	htt "net/http/httptest"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/store"
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

func (s *RESTSuite) TestStream(c *C) {
	// Make two accounts.
	user1 := &auth.Login{
		User:   users.User{Name: "bodie"},
		PWHash: []byte("hello"),
	}
	user2 := &auth.Login{
		User:   users.User{Name: "bob"},
		PWHash: []byte("12345"),
	}
	c.Assert(incept.Incept(s.tickets[0], user1, s.db), IsNil)
	c.Assert(incept.Incept(s.tickets[1], user2, s.db), IsNil)

	// Get a session token for each.
	sesh1, sesh2 := new(auth.Session), new(auth.Session)
	c.Assert(sgt.GetSession(user1.Name, sesh1, s.db), IsNil)
	c.Assert(sgt.GetSession(user2.Name, sesh2, s.db), IsNil)
	token1 := base64.RawURLEncoding.EncodeToString(sesh1.Token)
	token2 := base64.RawURLEncoding.EncodeToString(sesh2.Token)

	r := httprouter.New()
	c.Assert(rest.Stream{DB: s.db}.Bind(r), IsNil)
	// Make a testing server to run it.
	srv := htt.NewServer(r)
	defer srv.Close()

	id := uuid.NewV4()

	str := &stream.Stream{
		Owner:   "bodie",
		ID:      id.String(),
		Readers: map[string]bool{"bodie": true, "bob": true},
		Writers: map[string]bool{"bodie": true, "bob": true},
	}

	// Have to POST a new Stream.
	c.Assert(s.db.Update(stream.Upsert(str)), IsNil)

	// Get a new ws connection for each user.
	conn1, err := sgt.GetWSClient(token1, srv.URL+"/streams/"+str.ID+"/start")
	c.Assert(err, IsNil)
	conn1b, err := sgt.GetWSClient(token1, srv.URL+"/streams/"+str.ID+"/start")
	c.Assert(err, IsNil)
	conn2, err := sgt.GetWSClient(token2, srv.URL+"/streams/"+str.ID+"/start")
	c.Assert(err, IsNil)

	// Any sent message should be echoed to all receivers.
	c.Assert(ws.JSON.Send(conn1, &stream.Message{Content: "hello1"}), IsNil)
	c.Assert(ws.JSON.Send(conn1b, &stream.Message{Content: "hello1b"}), IsNil)
	c.Assert(ws.JSON.Send(conn2, &stream.Message{Content: "hello2"}), IsNil)

	seen := make(map[string]map[string]int)
	expect := map[string]map[string]int{
		"conn1":  {"hello1b": 1, "hello2": 1},
		"conn1b": {"hello1": 1, "hello2": 1},
		"conn2":  {"hello1b": 1, "hello1": 1},
	}

	msg := new(stream.Message)
	for _, conn := range []struct {
		name string
		*ws.Conn
	}{
		{"conn1", conn1}, {"conn1", conn1},
		{"conn1b", conn1b}, {"conn1b", conn1b},
		{"conn2", conn2}, {"conn2", conn2},
	} {
		c.Assert(ws.JSON.Receive(conn.Conn, msg), IsNil)
		if _, ok := seen[conn.name]; !ok {
			seen[conn.name] = make(map[string]int)
		}
		seen[conn.name][msg.Content]++
		c.Check(msg.Kind, Equals, 0)
	}

	c.Check(seen, DeepEquals, expect)

	conn1.Close()
	conn1b.Close()

	var surv river.Surveyor
	c.Assert(s.db.Update(func(tx *bolt.Tx) (e error) {
		surv, e = river.NewSurvey(tx,
			sgt.ShortWait,
			river.HangupBucket,
			store.Bucket(str.ID),
			store.Bucket("bob"),
		)
		return
	}), IsNil)

	c.Assert(river.MakeSurvey(surv, river.HUP, river.OK), IsNil)
	bs, err := ioutil.ReadAll(conn2)
	c.Check(string(bs), Equals, "stream hung up: EOF")
	c.Check(err, IsNil) // Note that nil = io.EOF in io.ReadAll.
}

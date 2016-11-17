package rest_test

import (
	"encoding/base64"
	"encoding/json"
	"net/http/httptest"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/notif"
	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream/river"
	sgt "github.com/synapse-garden/sg-proto/testing"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	"github.com/julienschmidt/httprouter"
	ws "golang.org/x/net/websocket"
	. "gopkg.in/check.v1"
)

var _ = rest.API(rest.Notif{})

type fooResourcer struct{ X int }

func (fooResourcer) Resource() store.Resource {
	return store.Resource("foo")
}

func (s *RESTSuite) TestConnectNotifs(c *C) {
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

	// Make two Publishers under bucket "notifs", and one under "brotifs".
	var pub1, pub2 river.Pub
	c.Assert(s.db.Update(func(tx *bolt.Tx) (err error) {
		pub1, err = river.NewPub("pub1", "notifs", tx)
		if err != nil {
			return
		}
		pub2, err = river.NewPub("pub2", "notifs", tx)
		if err != nil {
			return
		}
		// TODO: test other stream bucket not connected to
		// pub3, err = stream.NewPub("pub3", "brotifs", tx)
		// if err != nil {
		// 	return
		// }

		return
	}), IsNil)

	r := httprouter.New()
	c.Assert(rest.Notif{DB: s.db}.Bind(r), IsNil)
	// Make a testing server to run it.
	srv := httptest.NewServer(r)
	defer srv.Close()

	// Get a new ws connection for each user.  Topic is set to
	// notif.UserTopic based on token => username lookup.
	conn1 := getWSClient(c, token1, srv.URL+"/notifs")
	conn2 := getWSClient(c, token2, srv.URL+"/notifs")

	// Messages published on pub1 / pub2 are received for each user
	// correctly, but pub3 messages are not received since it wasn't
	// connected to.
	msg := new(store.ResourceBox)
	expect1, err := json.Marshal(fooResourcer{})
	c.Assert(err, IsNil)
	expect2, err := json.Marshal(fooResourcer{X: 1})
	c.Assert(err, IsNil)

	c.Assert(notif.Encode(pub1, fooResourcer{}, notif.MakeUserTopic("bodie")), IsNil)
	c.Check(ws.JSON.Receive(conn1, msg), IsNil)
	c.Check(msg.Contents, DeepEquals, expect1)
	c.Assert(notif.Encode(pub2, fooResourcer{X: 1}, notif.MakeUserTopic("bob")), IsNil)
	c.Check(ws.JSON.Receive(conn2, msg), IsNil)
	c.Check(msg.Contents, DeepEquals, expect2)

	c.Assert(conn1.Close(), IsNil)
	c.Assert(conn2.Close(), IsNil)
}

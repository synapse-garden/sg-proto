package rest_test

import (
	"encoding/base64"
	"net/http"
	htt "net/http/httptest"
	"net/url"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/stream"
	sgt "github.com/synapse-garden/sg-proto/testing"
	"github.com/synapse-garden/sg-proto/users"

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
	token1 := base64.StdEncoding.EncodeToString(sesh1.Token)
	token2 := base64.StdEncoding.EncodeToString(sesh2.Token)

	r := httprouter.New()
	c.Assert(rest.Stream(r, s.db), IsNil)
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
	conn1 := getWSClient(c, token1, srv.URL+"/streams/"+str.ID+"/start")
	conn2 := getWSClient(c, token2, srv.URL+"/streams/"+str.ID+"/start")

	// Any sent message should be echoed.
	c.Assert(ws.JSON.Send(conn1, &stream.Message{Content: "hello1"}), IsNil)
	c.Assert(ws.JSON.Send(conn2, &stream.Message{Content: "hello2"}), IsNil)

	msg := new(stream.Message)
	c.Assert(ws.JSON.Receive(conn1, msg), IsNil)
	c.Check(msg.Content, Equals, "hello2")
	c.Check(msg.Kind, Equals, 0)
	c.Assert(ws.JSON.Receive(conn2, msg), IsNil)
	c.Check(msg.Content, Equals, "hello1")
	c.Check(msg.Kind, Equals, 0)

	conn1.Close()
	conn2.Close()
}

func getWSClient(c *C, token, urlStr string) *ws.Conn {
	urlLoc, err := url.Parse(urlStr)
	c.Assert(err, IsNil)
	urlLoc.Scheme = "ws"

	conn, err := ws.DialConfig(&ws.Config{
		Location: urlLoc,
		Origin:   &url.URL{},
		Version:  ws.ProtocolVersionHybi13,
		Header: http.Header{
			"Authorization": {"Bearer " + token},
		},
	})
	c.Assert(err, IsNil)

	return conn
}

package river_test

import (
	"reflect"

	"github.com/synapse-garden/sg-proto/stream/river"

	"github.com/boltdb/bolt"
	"github.com/go-mangos/mangos/protocol/sub"
	. "gopkg.in/check.v1"
)

func makeSub(c *C,
	db *bolt.DB,
	streamID string,
	topics ...river.Topic,
) (r river.Sub) {
	c.Assert(db.Update(func(tx *bolt.Tx) (e error) {
		r, e = river.NewSub(streamID, tx, topics...)
		return
	}), IsNil)

	return
}

func (s *RiverSuite) TestNewSub(c *C) {
	p1 := makePub(c, s.db, "p1", "goodbye")
	p2 := makePub(c, s.db, "p2", "goodbye")

	checkRivers(c, s.db, "goodbye", "p1", "p2")

	var sbGlob, sbHello, sbMore river.Sub
	c.Assert(s.db.Update(func(tx *bolt.Tx) (e error) {
		sbGlob, e = river.NewSub("goodbye", tx)
		return
	}), IsNil)

	subsock, err := sub.NewSocket()
	c.Assert(err, IsNil)
	expect := river.MakeSub(subsock)

	c.Check(reflect.TypeOf(sbGlob), Equals, reflect.TypeOf(expect))

	helloTopic := testingTopic{pre: 0x1, code: []byte("hello")}
	goodbyeTopic := testingTopic{pre: 0x2, code: []byte("goodbye")}

	c.Assert(s.db.Update(func(tx *bolt.Tx) (e error) {
		sbHello, e = river.NewSub("goodbye", tx, helloTopic)
		return
	}), IsNil)

	c.Check(reflect.TypeOf(sbHello), Equals, reflect.TypeOf(expect))

	msgsGlob, errsGlob := startRecving(sbGlob)
	msgsHello, errsHello := startRecving(sbHello)

	c.Assert(p1.Send(river.BytesFor(river.Global, []byte("hello1"))), IsNil)
	checkMessagesRecvd(c, msgsGlob, errsGlob, "hello1")
	tryNotRecv(c, msgsHello, errsHello)

	c.Assert(p2.Send(river.BytesFor(helloTopic, []byte("goodbye1"))), IsNil)
	checkMessagesRecvd(c, msgsHello, errsGlob, "goodbye1")
	tryNotRecv(c, msgsGlob, errsGlob)

	c.Assert(s.db.Update(func(tx *bolt.Tx) (e error) {
		sbMore, e = river.NewSub("goodbye", tx, helloTopic, goodbyeTopic)
		return
	}), IsNil)

	c.Check(reflect.TypeOf(sbHello), Equals, reflect.TypeOf(expect))

	msgsMore, errsMore := startRecving(sbMore)

	c.Assert(p1.Send(river.BytesFor(helloTopic, []byte("hello3"))), IsNil)
	tryNotRecv(c, msgsGlob, errsGlob)
	checkMessagesRecvd(c, msgsHello, errsHello, "hello3")
	checkMessagesRecvd(c, msgsMore, errsMore, "hello3")

	c.Assert(p1.Send(river.BytesFor(river.Global, []byte("fresh3"))), IsNil)
	checkMessagesRecvd(c, msgsGlob, errsGlob, "fresh3")
	tryNotRecv(c, msgsHello, errsHello)
	tryNotRecv(c, msgsMore, errsMore)

	c.Assert(sbGlob.Close(), IsNil)
	c.Assert(sbHello.Close(), IsNil)
	c.Assert(sbMore.Close(), IsNil)

	c.Assert(p1.Close(), IsNil)
	c.Assert(p2.Close(), IsNil)

	c.Check(tryRecvError(c, msgsGlob, errsGlob), ErrorMatches, "connection closed")
	c.Check(tryRecvError(c, msgsHello, errsHello), ErrorMatches, "connection closed")
	c.Check(tryRecvError(c, msgsMore, errsMore), ErrorMatches, "connection closed")
}

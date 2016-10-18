package river_test

import (
	"reflect"

	"github.com/synapse-garden/sg-proto/stream/river"

	"github.com/boltdb/bolt"
	"github.com/go-mangos/mangos/protocol/bus"
	. "gopkg.in/check.v1"
)

func makePub(c *C, db *bolt.DB, id, streamID string) (r river.Pub) {
	c.Assert(db.Update(func(tx *bolt.Tx) (e error) {
		r, e = river.NewPub(id, streamID, tx)
		return
	}), IsNil)

	return
}

func (s *RiverSuite) TestNewPub(c *C) {
	var p1 river.Pub
	c.Assert(s.db.Update(func(tx *bolt.Tx) (e error) {
		p1, e = river.NewPub("p1", "goodbye", tx)
		return
	}), IsNil)
	expect, err := bus.NewSocket()
	c.Assert(err, IsNil)

	c.Check(reflect.TypeOf(p1), Equals, reflect.TypeOf(expect))

	checkRivers(c, s.db, "goodbye", "p1")

	var p2 river.Pub
	c.Assert(s.db.Update(func(tx *bolt.Tx) (e error) {
		p2, e = river.NewPub("p2", "goodbye", tx)
		return
	}), IsNil)
	expect, err = bus.NewSocket()
	c.Assert(err, IsNil)

	c.Check(reflect.TypeOf(p2), Equals, reflect.TypeOf(expect))

	checkRivers(c, s.db, "goodbye", "p1", "p2")

	c.Assert(s.db.Update(func(tx *bolt.Tx) (e error) {
		_, e = river.NewPub("p1", "goodbye", tx)
		return
	}), ErrorMatches, "river `p1` already exists")

	checkRivers(c, s.db, "goodbye", "p1", "p2")

	sb1 := makeSub(c, s.db, "goodbye")
	msgs1, errs1 := startRecving(sb1)

	c.Assert(p1.Send(river.BytesFor(river.Global, []byte("hello1"))), IsNil)
	checkMessagesRecvd(c, msgs1, errs1, "hello1")

	c.Assert(p2.Send(river.BytesFor(river.Global, []byte("hello2"))), IsNil)
	checkMessagesRecvd(c, msgs1, errs1, "hello2")

	sb2 := makeSub(c, s.db, "goodbye")
	msgs2, errs2 := startRecving(sb2)

	c.Assert(p1.Send(river.BytesFor(river.Global, []byte("hello3"))), IsNil)
	checkMessagesRecvd(c, msgs1, errs1, "hello3")
	checkMessagesRecvd(c, msgs2, errs2, "hello3")

	c.Assert(sb1.Close(), IsNil)
	c.Assert(sb2.Close(), IsNil)
	c.Assert(p1.Close(), IsNil)
	c.Assert(p2.Close(), IsNil)

	c.Check(tryRecvError(c, msgs1, errs1), ErrorMatches, "connection closed")
	c.Check(tryRecvError(c, msgs2, errs2), ErrorMatches, "connection closed")
}

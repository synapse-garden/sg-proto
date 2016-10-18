package river_test

import (
	"reflect"

	"github.com/synapse-garden/sg-proto/stream/river"

	"github.com/boltdb/bolt"
	"github.com/go-mangos/mangos/protocol/bus"
	. "gopkg.in/check.v1"
)

var _ = river.River(&mockRiver{})

func makeBus(c *C, db *bolt.DB, id, streamID string) (r river.River) {
	c.Assert(db.Update(func(tx *bolt.Tx) (e error) {
		r, e = river.NewBus(id, streamID, tx)
		return
	}), IsNil)

	return
}

func (s *RiverSuite) TestNewBus(c *C) {
	var r1 river.River
	c.Check(s.db.Update(func(tx *bolt.Tx) (e error) {
		r1, e = river.NewBus("hello", "goodbye", tx)
		return
	}), IsNil)

	expect, err := bus.NewSocket()
	c.Assert(err, IsNil)
	c.Check(reflect.TypeOf(r1), Equals, reflect.TypeOf(expect))

	checkRivers(c, s.db, "goodbye", "hello")

	c.Check(r1.Send([]byte("hello0")), IsNil)

	err = s.db.Update(func(tx *bolt.Tx) (e error) {
		_, e = river.NewBus("hello", "goodbye", tx)
		return
	})

	c.Assert(err, DeepEquals, river.MakeRiverExistsErr("hello"))
	checkRivers(c, s.db, "goodbye", "hello")

	var r2 river.River
	c.Check(s.db.Update(func(tx *bolt.Tx) (e error) {
		r2, e = river.NewBus("hello2", "goodbye", tx)
		return
	}), IsNil)

	c.Assert(r1.Send([]byte("hello1")), IsNil)
	c.Assert(r2.Send([]byte("hello2")), IsNil)

	r1msgs, r1errs := startRecving(r1)
	r2msgs, r2errs := startRecving(r2)

	checkMessagesRecvd(c, r1msgs, r1errs, "hello2")
	checkMessagesRecvd(c, r2msgs, r2errs, "hello1")

	c.Assert(s.db.Update(river.DeleteRiver("hello2", "goodbye")), IsNil)
	r3 := makeBus(c, s.db, "hello3", "goodbye")

	c.Assert(r1.Send([]byte("hello1")), IsNil)
	c.Assert(r2.Send([]byte("hello2")), IsNil)
	c.Assert(r3.Send([]byte("hello3")), IsNil)
	r3msgs, r3errs := startRecving(r3)

	checkMessagesRecvd(c, r1msgs, r1errs, "hello2", "hello3")
	checkMessagesRecvd(c, r2msgs, r2errs, "hello1")
	checkMessagesRecvd(c, r3msgs, r3errs, "hello1")

	c.Assert(r1.Close(), IsNil)
	c.Assert(r2.Close(), IsNil)
	c.Assert(r3.Close(), IsNil)

	c.Check(tryRecvError(c, r1msgs, r1errs), ErrorMatches, "connection closed")
	c.Check(tryRecvError(c, r2msgs, r2errs), ErrorMatches, "connection closed")
	c.Check(tryRecvError(c, r3msgs, r3errs), ErrorMatches, "connection closed")
}

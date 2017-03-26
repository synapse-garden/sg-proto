package river_test

import (
	"reflect"
	"strconv"

	"github.com/pkg/errors"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream/river"

	"github.com/boltdb/bolt"
	"github.com/go-mangos/mangos/protocol/bus"
	. "gopkg.in/check.v1"
)

var _ = river.River(&mockRiver{})

func makeBus(c *C, db *bolt.DB, id, streamID string) (r river.Bus) {
	c.Assert(db.Update(func(tx *bolt.Tx) (e error) {
		r, e = river.NewBus(id, streamID, tx)
		return
	}), IsNil)

	return
}

func checkBuses(c *C, db *bolt.DB, streamID string, rivers map[string][]uint64) {
	seen := make(map[string][]uint64)
	c.Check(db.View(func(tx *bolt.Tx) error {
		b, err := store.GetNestedBucket(
			tx.Bucket(river.RiverBucket),
			[]byte(streamID),
		)
		if err != nil {
			return errors.Wrap(err, "failed to get nested bucket")
		}
		return b.ForEach(func(k, _ []byte) error {
			bucket := string(k)
			seen[bucket] = []uint64{}
			return b.Bucket(k).ForEach(func(k2, _ []byte) error {
				id, err := strconv.ParseUint(string(k2), 10, 64)
				if err != nil {
					return err
				}
				seen[bucket] = append(seen[bucket], id)
				return nil
			})
		})
	}), IsNil)

	c.Check(seen, DeepEquals, rivers)
}

func (s *RiverSuite) TestNewBus(c *C) {
	c.Log("create a bus in /goodbye/hello")
	var r1 river.Bus
	c.Check(s.db.Update(func(tx *bolt.Tx) (e error) {
		r1, e = river.NewBus("hello", "goodbye", tx)
		return
	}), IsNil)

	inner, err := bus.NewSocket()
	c.Assert(err, IsNil)
	expect := river.MakeBus(inner, 1)
	c.Log("newly made bus should have ID 1")
	c.Check(reflect.TypeOf(r1), Equals, reflect.TypeOf(expect))
	c.Check(r1.ID(), Equals, uint64(1))

	c.Log("one bus should exist: /goodbye/hello/1")
	checkBuses(c, s.db, "goodbye", map[string][]uint64{"hello": {1}})

	c.Log("Nothing happens if you send bytes on it.")
	c.Check(r1.Send([]byte("hello0")), IsNil)

	c.Log("Creating a new Bus in that bucket doesn't cause a problem.")
	var r1b river.Bus
	err = s.db.Update(func(tx *bolt.Tx) (e error) {
		r1b, e = river.NewBus("hello", "goodbye", tx)
		return
	})

	c.Log("Its ID will be 2")
	c.Check(reflect.TypeOf(r1), Equals, reflect.TypeOf(expect))
	c.Check(r1b.ID(), Equals, uint64(2))

	c.Log("Now buses 1 and 2 exist in /goodbye/hello")
	checkBuses(c, s.db, "goodbye", map[string][]uint64{"hello": {1, 2}})

	c.Log("Closing one and deleting it doesn't cause a problem")
	c.Assert(s.db.Update(func(tx *bolt.Tx) error {
		if err := r1b.Close(); err != nil {
			return err
		}
		return river.DeleteBus("hello", "goodbye", r1b.ID())(tx)
	}), IsNil)

	c.Log("Making a new bus in /goodbye/hello2 works fine")
	var r2 river.Bus
	c.Check(s.db.Update(func(tx *bolt.Tx) (e error) {
		r2, e = river.NewBus("hello2", "goodbye", tx)
		return
	}), IsNil)

	c.Log("Its ID will be 1")
	c.Check(r2.ID(), Equals, uint64(1))

	c.Log("Now two buses exist: /goodbye/hello/1 and /goodbye/hello2/1")
	checkBuses(c, s.db, "goodbye", map[string][]uint64{
		"hello":  {1},
		"hello2": {1},
	})

	r1msgs, r1errs := startRecving(r1)
	r2msgs, r2errs := startRecving(r2)

	c.Log("Sending a message on each bus works")
	c.Assert(r1.Send([]byte("hello1")), IsNil)
	c.Assert(r2.Send([]byte("hello2")), IsNil)

	checkMessagesRecvd(c, r1msgs, r1errs, "hello2")
	checkMessagesRecvd(c, r2msgs, r2errs, "hello1")

	c.Log("Now delete one from the DB but leave it listening")
	c.Log("New buses won't dial it, so it won't receive their msgs")
	c.Assert(s.db.Update(river.DeleteBus("hello2", "goodbye", r2.ID())), IsNil)

	c.Log("Now only /goodbye/hello/1 remains")
	checkBuses(c, s.db, "goodbye", map[string][]uint64{
		"hello":  {1},
		"hello2": {},
	})

	c.Log("Making a new bus in hello3 works")
	r3 := makeBus(c, s.db, "hello3", "goodbye")

	c.Log("New bus in hello3 has ID 1")
	c.Check(r3.ID(), Equals, uint64(1))

	c.Log("Now Buses /goodbye/hello/1 and /goodbye/hello3/1 exist")
	checkBuses(c, s.db, "goodbye", map[string][]uint64{
		"hello":  {1},
		"hello2": {},
		"hello3": {1},
	})

	r3msgs, r3errs := startRecving(r3)

	c.Assert(r1.Send([]byte("hello1")), IsNil)
	c.Assert(r2.Send([]byte("hello2")), IsNil)
	c.Assert(r3.Send([]byte("hello3")), IsNil)

	c.Log("bus /goodbye/hello/1 receives msgs from hello2 and hello3")
	checkMessagesRecvd(c, r1msgs, r1errs, "hello2", "hello3")
	c.Log("bus /goodbye/hello2/1 only receives msgs from old conn")
	checkMessagesRecvd(c, r2msgs, r2errs, "hello1")
	c.Log("bus /goodbye/hello3/1 only receives msgs from new conn")
	checkMessagesRecvd(c, r3msgs, r3errs, "hello1")

	c.Assert(r1.Close(), IsNil)
	c.Assert(r2.Close(), IsNil)
	c.Assert(r3.Close(), IsNil)

	c.Check(tryRecvError(c, r1msgs, r1errs), ErrorMatches, "connection closed")
	c.Check(tryRecvError(c, r2msgs, r2errs), ErrorMatches, "connection closed")
	c.Check(tryRecvError(c, r3msgs, r3errs), ErrorMatches, "connection closed")
}

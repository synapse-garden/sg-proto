package stream_test

import (
	"errors"
	"reflect"
	"time"

	"github.com/synapse-garden/sg-proto/stream"
	"github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
	"github.com/go-mangos/mangos/protocol/bus"
	"github.com/go-mangos/mangos/protocol/sub"
	. "gopkg.in/check.v1"
)

type sender interface {
	Send([]byte) error
}
type recver interface {
	Recv() ([]byte, error)
}

func checkRivers(c *C, db *bolt.DB, streamID string, rivers ...string) {
	expect := make(map[string]int)
	for _, r := range rivers {
		expect[r]++
	}
	seen := make(map[string]int)
	c.Check(db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(stream.RiverBucket).Bucket([]byte(streamID))
		if b == nil {
			return errors.New("expected missing Stream bucket %#q")
		}
		c := b.Cursor()

		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			seen[string(k)]++
		}

		return nil
	}), IsNil)

	c.Check(seen, DeepEquals, expect)
}

func makeBus(c *C, db *bolt.DB, id, streamID string) (r stream.River) {
	c.Assert(db.Update(func(tx *bolt.Tx) (e error) {
		r, e = stream.NewBus(id, streamID, tx)
		return
	}), IsNil)

	return
}

func makePub(c *C, db *bolt.DB, id, streamID string) (r stream.River) {
	c.Assert(db.Update(func(tx *bolt.Tx) (e error) {
		r, e = stream.NewPub(id, streamID, tx)
		return
	}), IsNil)

	return
}

func makeSub(c *C,
	db *bolt.DB,
	streamID string,
	topics ...stream.Topic,
) (r stream.River) {
	c.Assert(db.Update(func(tx *bolt.Tx) (e error) {
		r, e = stream.NewSub(streamID, tx, topics...)
		return
	}), IsNil)

	return
}

type testingTopic []byte

func (t testingTopic) Code() []byte {
	return []byte(t)
}

func (t testingTopic) Name() string {
	return "testing"
}

func recvTilDone(c *C,
	r recver,
	m chan<- []byte,
	done <-chan struct{},
	errs chan<- error,
) {
	for {
		select {
		case <-done:
			return
		default:
		}

		bs, err := r.Recv()
		if err != nil {
			errs <- err
			return
		}
		m <- bs
	}
}

func tryRecv(c *C, m <-chan []byte, errs <-chan error) []byte {
	select {
	case got := <-m:
		return got
	case err := <-errs:
		c.Logf("unexpected error received: %#v", err.Error())
		c.FailNow()
	case <-time.After(testing.ShortWait):
		c.Logf("failed to receive message after %s timeout",
			testing.ShortWait.String())
		c.FailNow()
	}
	return nil
}

func tryNotRecv(c *C, m <-chan []byte, errs <-chan error) {
	select {
	case msg := <-m:
		c.Logf("received unexpected message %#q", msg)
		c.FailNow()
	case err := <-errs:
		c.Logf("unexpected error received: %#v", err.Error())
		c.FailNow()
	case <-time.After(3 * testing.ShortWait):
	}
}

func tryRecvError(c *C, m <-chan []byte, errs <-chan error) error {
	select {
	case msg := <-m:
		c.Logf("received unexpected message %#q", msg)
		c.FailNow()
	case err := <-errs:
		return err
	case <-time.After(3 * testing.ShortWait):
		c.Logf("failed to receive error after %s timeout",
			testing.ShortWait.String())
		c.FailNow()
	}
	return nil
}

func checkMessagesRecvd(c *C,
	m <-chan []byte,
	errs <-chan error,
	msgs ...string,
) {
	expect := make(map[string]int)
	got := make(map[string]int)
	for _, m := range msgs {
		expect[m]++
	}

	for i := 0; i < len(msgs); i++ {
		got[string(tryRecv(c, m, errs))]++
	}
	tryNotRecv(c, m, errs)
}

func startRecving(r recver) (<-chan []byte, <-chan error) {
	msgs, errs := make(chan []byte), make(chan error)
	go func() {
		for {
			msg, err := r.Recv()
			if err != nil {
				errs <- err
				return
			}
			msgs <- msg
		}
	}()

	return msgs, errs
}

func (s *StreamSuite) TestNewPub(c *C) {
	var p1 stream.River
	c.Assert(s.db.Update(func(tx *bolt.Tx) (e error) {
		p1, e = stream.NewPub("p1", "goodbye", tx)
		return
	}), IsNil)
	expect, err := bus.NewSocket()
	c.Assert(err, IsNil)

	c.Check(reflect.TypeOf(p1), Equals, reflect.TypeOf(expect))

	checkRivers(c, s.db, "goodbye", "p1")

	var p2 stream.River
	c.Assert(s.db.Update(func(tx *bolt.Tx) (e error) {
		p2, e = stream.NewPub("p2", "goodbye", tx)
		return
	}), IsNil)
	expect, err = bus.NewSocket()
	c.Assert(err, IsNil)

	c.Check(reflect.TypeOf(p2), Equals, reflect.TypeOf(expect))

	checkRivers(c, s.db, "goodbye", "p1", "p2")

	c.Assert(s.db.Update(func(tx *bolt.Tx) (e error) {
		_, e = stream.NewPub("p1", "goodbye", tx)
		return
	}), ErrorMatches, "river `p1` already exists")

	checkRivers(c, s.db, "goodbye", "p1", "p2")

	sb1 := makeSub(c, s.db, "goodbye")
	msgs1, errs1 := startRecving(sb1)

	c.Assert(p1.Send([]byte("hello1")), IsNil)
	checkMessagesRecvd(c, msgs1, errs1, "hello1")

	c.Assert(p2.Send([]byte("hello2")), IsNil)
	checkMessagesRecvd(c, msgs1, errs1, "hello2")

	sb2 := makeSub(c, s.db, "goodbye")
	msgs2, errs2 := startRecving(sb2)

	c.Assert(p1.Send([]byte("hello3")), IsNil)
	checkMessagesRecvd(c, msgs1, errs1, "hello3")
	checkMessagesRecvd(c, msgs2, errs2, "hello3")

	c.Assert(sb1.Close(), IsNil)
	c.Assert(sb2.Close(), IsNil)
	c.Assert(p1.Close(), IsNil)
	c.Assert(p2.Close(), IsNil)

	c.Check(tryRecvError(c, msgs1, errs1), ErrorMatches, "connection closed")
	c.Check(tryRecvError(c, msgs2, errs2), ErrorMatches, "connection closed")
}

func (s *StreamSuite) TestNewSub(c *C) {
	p1 := makePub(c, s.db, "p1", "goodbye")
	p2 := makePub(c, s.db, "p2", "goodbye")

	checkRivers(c, s.db, "goodbye", "p1", "p2")

	var sb, sbTop, sbMore stream.River
	c.Assert(s.db.Update(func(tx *bolt.Tx) (e error) {
		sb, e = stream.NewSub("goodbye", tx)
		return
	}), IsNil)

	expect, err := sub.NewSocket()
	c.Assert(err, IsNil)

	c.Check(reflect.TypeOf(sb), Equals, reflect.TypeOf(expect))

	c.Assert(s.db.Update(func(tx *bolt.Tx) (e error) {
		sbTop, e = stream.NewSub("goodbye", tx, testingTopic("hello"))
		return
	}), IsNil)

	c.Check(reflect.TypeOf(sbTop), Equals, reflect.TypeOf(expect))

	msgs, errs := startRecving(sb)
	msgsTop, errsTop := startRecving(sbTop)

	c.Assert(p1.Send([]byte("hello1")), IsNil)
	checkMessagesRecvd(c, msgs, errs, "hello1")
	checkMessagesRecvd(c, msgsTop, errsTop, "hello1")

	c.Assert(p2.Send([]byte("goodbye1")), IsNil)
	checkMessagesRecvd(c, msgs, errs, "goodbye1")
	tryNotRecv(c, msgsTop, errsTop)

	c.Assert(s.db.Update(func(tx *bolt.Tx) (e error) {
		sbMore, e = stream.NewSub("goodbye", tx, testingTopic("hello"), testingTopic("goodbye"))
		return
	}), IsNil)

	c.Check(reflect.TypeOf(sbTop), Equals, reflect.TypeOf(expect))

	msgsMore, errsMore := startRecving(sbMore)

	c.Assert(p1.Send([]byte("hello3")), IsNil)
	checkMessagesRecvd(c, msgs, errs, "hello3")
	checkMessagesRecvd(c, msgsTop, errsTop, "hello3")
	checkMessagesRecvd(c, msgsMore, errsMore, "hello3")

	c.Assert(p1.Send([]byte("fresh3")), IsNil)
	checkMessagesRecvd(c, msgs, errs, "fresh3")
	tryNotRecv(c, msgsTop, errsTop)
	tryNotRecv(c, msgsMore, errsMore)

	c.Assert(sb.Close(), IsNil)
	c.Assert(sbTop.Close(), IsNil)
	c.Assert(sbMore.Close(), IsNil)

	c.Assert(p1.Close(), IsNil)
	c.Assert(p2.Close(), IsNil)

	c.Check(tryRecvError(c, msgs, errs), ErrorMatches, "connection closed")
	c.Check(tryRecvError(c, msgsTop, errsTop), ErrorMatches, "connection closed")
	c.Check(tryRecvError(c, msgsMore, errsMore), ErrorMatches, "connection closed")
}

func (s *StreamSuite) TestNewBus(c *C) {
	var r1 stream.River
	c.Check(s.db.Update(func(tx *bolt.Tx) (e error) {
		r1, e = stream.NewBus("hello", "goodbye", tx)
		return
	}), IsNil)

	expect, err := bus.NewSocket()
	c.Assert(err, IsNil)
	c.Check(reflect.TypeOf(r1), Equals, reflect.TypeOf(expect))

	checkRivers(c, s.db, "goodbye", "hello")

	c.Check(r1.Send([]byte("hello0")), IsNil)

	err = s.db.Update(func(tx *bolt.Tx) (e error) {
		_, e = stream.NewBus("hello", "goodbye", tx)
		return
	})

	c.Assert(err, DeepEquals, stream.MakeRiverExistsErr("hello"))
	checkRivers(c, s.db, "goodbye", "hello")

	var r2 stream.River
	c.Check(s.db.Update(func(tx *bolt.Tx) (e error) {
		r2, e = stream.NewBus("hello2", "goodbye", tx)
		return
	}), IsNil)

	c.Assert(r1.Send([]byte("hello1")), IsNil)
	c.Assert(r2.Send([]byte("hello2")), IsNil)

	r1msgs, r1errs := startRecving(r1)
	r2msgs, r2errs := startRecving(r2)

	checkMessagesRecvd(c, r1msgs, r1errs, "hello2")
	checkMessagesRecvd(c, r2msgs, r2errs, "hello1")

	c.Assert(s.db.Update(stream.DeleteRiver("hello2", "goodbye")), IsNil)
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

func (s *StreamSuite) TestCheckRiverNotExists(c *C) {
	err := s.db.View(stream.CheckRiverNotExists("hello", "goodbye"))
	c.Check(err, IsNil)

	r := makeBus(c, s.db, "hello", "goodbye")
	defer func() { c.Assert(r.Close(), IsNil) }()

	err = s.db.View(stream.CheckRiverNotExists("hello", "goodbye"))
	c.Check(err, DeepEquals, stream.MakeRiverExistsErr("hello"))
}

func (s *StreamSuite) TestClearRivers(c *C) {
	c.Check(s.db.Update(stream.ClearRivers), IsNil)

	c.Assert(makeBus(c, s.db, "hello1", "goodbye").Close(), IsNil)
	c.Assert(makeBus(c, s.db, "hello2", "goodbye").Close(), IsNil)

	checkRivers(c, s.db, "goodbye", "hello1", "hello2")

	c.Check(s.db.Update(stream.ClearRivers), IsNil)

	c.Assert(s.db.View(func(tx *bolt.Tx) error {
		cr := tx.Bucket(stream.RiverBucket).Cursor()
		for k, _ := cr.First(); k != nil; k, _ = cr.Next() {
			c.Logf("  unexpected bucket found: %#q", k)
			c.FailNow()
		}

		return nil
	}), IsNil)
}

package river_test

import (
	"errors"
	"time"

	"github.com/synapse-garden/sg-proto/stream/river"
	"github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
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
		b := tx.Bucket(river.RiverBucket).Bucket([]byte(streamID))
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

type mockRiver struct {
	SendErrs []error
	Sent     [][]byte
	Recvs    [][]byte
	RecvErrs []error
	closeErr error

	rCount int
}

func (m *mockRiver) Send(msg []byte) error {
	m.Sent = append(m.Sent, msg)
	return m.SendErrs[len(m.Sent)-1]
}

func (m *mockRiver) Recv() ([]byte, error) {
	defer func() { m.rCount++ }()
	return m.Recvs[m.rCount], m.RecvErrs[m.rCount]
}

func (m *mockRiver) Close() error { return m.closeErr }

type testingTopic struct {
	pre  byte
	code []byte
}

func (t testingTopic) Prefix() byte { return t.pre }

func (t testingTopic) Code() []byte {
	return append([]byte{t.pre}, t.code...)
}

func (t testingTopic) Name() string {
	return "testing"
}

func (t testingTopic) Len() int {
	return len(t.code) + 1
}

var (
	_ = river.Sub(&mockRiver{})
	_ = river.Pub(&mockRiver{})
	_ = river.Topic(testingTopic{})
)

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

func (s *RiverSuite) TestCheckRiverNotExists(c *C) {
	err := s.db.View(river.CheckRiverNotExists("hello", "goodbye"))
	c.Check(err, IsNil)

	r := makeBus(c, s.db, "hello", "goodbye")
	defer func() { c.Assert(r.Close(), IsNil) }()

	err = s.db.View(river.CheckRiverNotExists("hello", "goodbye"))
	c.Check(err, DeepEquals, river.MakeRiverExistsErr("hello"))
}

func (s *RiverSuite) TestClearRivers(c *C) {
	c.Check(s.db.Update(river.ClearRivers), IsNil)

	c.Assert(makeBus(c, s.db, "hello1", "goodbye").Close(), IsNil)
	c.Assert(makeBus(c, s.db, "hello2", "goodbye").Close(), IsNil)

	checkRivers(c, s.db, "goodbye", "hello1", "hello2")

	c.Check(s.db.Update(river.ClearRivers), IsNil)

	c.Assert(s.db.View(func(tx *bolt.Tx) error {
		cr := tx.Bucket(river.RiverBucket).Cursor()
		for k, _ := cr.First(); k != nil; k, _ = cr.Next() {
			c.Logf("  unexpected bucket found: %#q", k)
			c.FailNow()
		}

		return nil
	}), IsNil)
}

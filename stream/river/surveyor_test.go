package river_test

import (
	"time"

	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream/river"
	"github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
	"github.com/go-mangos/mangos"
	. "gopkg.in/check.v1"
)

type testResponder struct {
	mangos.Socket
	id uint64
}

func (t testResponder) ID() uint64 { return t.id }

var _ = river.Responder(testResponder{})
var _ = river.Missing(river.MakeMissing([]uint64{0, 1, 2}))

const retryTime = 10 * time.Millisecond

func (s *RiverSuite) TestNewSurvey(c *C) {
	var rsp1, rsp2, rsp3 river.Responder

	// Create some artificial responders in the "user" bucket.
	c.Assert(s.db.Update(func(tx *bolt.Tx) (e error) {
		rsp1, e = river.NewResponder(tx,
			store.Bucket("surv"),
			store.Bucket("user"),
		)
		if e != nil {
			return
		}

		rsp2, e = river.NewResponder(tx,
			store.Bucket("surv"),
			store.Bucket("user"),
		)
		if e != nil {
			return
		}

		rsp3, e = river.NewResponder(tx,
			store.Bucket("surv"),
			store.Bucket("user"),
		)

		return
	}), IsNil)

	r1, r2, r3 := make(chan error), make(chan error), make(chan error)
	go func() {
		e := river.AwaitHangup(rsp1)
		r1 <- e
	}()
	go func() {
		e := river.AwaitHangup(rsp2)
		r2 <- e
	}()
	go func() {
		e := river.AwaitHangup(rsp3)
		r3 <- e
	}()

	var surv river.Surveyor
	err := s.db.View(func(tx *bolt.Tx) (e error) {
		surv, e = river.NewSurvey(tx,
			retryTime,
			[]byte("foo"), []byte("bar"),
		)
		return
	})

	c.Check(err, ErrorMatches, "no such stream `foo`")
	c.Check(river.IsStreamMissing(err), Equals, true)

	c.Assert(s.db.View(func(tx *bolt.Tx) (e error) {
		surv, e = river.NewSurvey(tx,
			retryTime,
			[]byte("surv"), []byte("user"),
		)
		return
	}), IsNil)

	c.Check(surv.Expect(), DeepEquals, map[uint64]bool{
		1: true, 2: true, 3: true,
	})

	c.Assert(river.MakeSurvey(surv, river.HUP, river.OK), IsNil)

	// Make sure a nil error was received for each respondent Send.
	for _, ch := range []chan error{r1, r2, r3} {
		select {
		case err := <-ch:
			c.Assert(err, IsNil)
		case <-time.After(testing.LongWait):
			c.Logf("failed to receive expected value after %s",
				testing.LongWait.String())
			c.FailNow()
		}
	}

	c.Assert(surv.Close(), IsNil)
	c.Assert(rsp1.Close(), IsNil)
	c.Assert(rsp2.Close(), IsNil)
	c.Assert(rsp3.Close(), IsNil)
}

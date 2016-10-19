package river_test

import (
	"time"

	"github.com/synapse-garden/sg-proto/stream/river"
	"github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
	"github.com/go-mangos/mangos"
	"github.com/go-mangos/mangos/protocol/surveyor"
	"github.com/go-mangos/mangos/transport/inproc"
	. "gopkg.in/check.v1"
)

type testSurveyor struct {
	mangos.Socket

	clients map[uint64]bool
}

func (t testSurveyor) Expect() map[uint64]bool { return t.clients }

var _ = river.Surveyor(testSurveyor{})

func (s *RiverSuite) TestNewResp(c *C) {
	var rsp1 river.Responder
	c.Assert(s.db.Update(func(tx *bolt.Tx) (err error) {
		rsp1, err = river.NewResponder(tx,
			[]byte("surv"),
			[]byte("bob"),
			[]byte("stuff"),
		)
		return
	}), IsNil)

	var rsp2 river.Responder
	c.Assert(s.db.Update(func(tx *bolt.Tx) (err error) {
		rsp2, err = river.NewResponder(tx,
			[]byte("surv"),
			[]byte("bob"),
			[]byte("stuff"),
		)
		return
	}), IsNil)

	var rsp3 river.Responder
	c.Assert(s.db.Update(func(tx *bolt.Tx) (err error) {
		rsp3, err = river.NewResponder(tx,
			[]byte("surv"),
			[]byte("bob"),
			[]byte("stuff"),
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
		// To generate an error:
		// idBs := make([]byte, 8)
		// binary.LittleEndian.PutUint64(idBs, rsp3.ID())
		// e := river.Wait(rsp3,
		// 	river.HUP,
		// 	river.Response(append(river.HUP, idBs...)),
		// 	append(river.UNKNOWN, idBs...),
		// )
	}()

	surv, err := surveyor.NewSocket()
	c.Assert(err, IsNil)

	surv.AddTransport(inproc.NewTransport())
	c.Assert(surv.SetOption(
		mangos.OptionSurveyTime, testing.ShortWait,
	), IsNil)
	c.Assert(surv.Dial("inproc://surv/bob/stuff/1"), IsNil)
	c.Assert(surv.Dial("inproc://surv/bob/stuff/2"), IsNil)
	c.Assert(surv.Dial("inproc://surv/bob/stuff/3"), IsNil)

	c.Assert(river.MakeSurvey(
		testSurveyor{surv, map[uint64]bool{
			1: true,
			2: true,
			3: true,
		}},
		river.HUP,
		river.OK,
	), IsNil)

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

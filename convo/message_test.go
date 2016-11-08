package convo_test

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/synapse-garden/sg-proto/convo"
	"github.com/synapse-garden/sg-proto/store"

	"github.com/boltdb/bolt"
	. "gopkg.in/check.v1"
)

func prepareMessages(c *C,
	db *bolt.DB,
	start, end time.Time,
	diff time.Duration,
) []convo.Message {
	var msgs []convo.Message
	c.Assert(db.Update(func(tx *bolt.Tx) error {
		b, err := store.MakeNestedBucket(
			tx.Bucket(convo.MessageBucket),
			store.Bucket("hello"),
		)
		c.Assert(err, IsNil)

		for i := 0; i < 10; i++ {
			stamp := start.Add(diff * time.Duration(i))
			msg := convo.Message{
				Sender:    "bob",
				Timestamp: stamp,
				Content:   "hello" + strconv.Itoa(i),
			}
			msgs = append(msgs, msg)
			msgBs, err := json.Marshal(msg)
			c.Assert(err, IsNil)
			c.Assert(b.Put(stamp.AppendFormat(
				nil, time.RFC3339,
			), msgBs), IsNil)
		}
		return nil
	}), IsNil)

	return msgs
}

func (s *ConvoSuite) TestGetMessageRange(c *C) {
	tStart, _ := time.Parse(time.Kitchen, "2:00AM")
	tEnd, _ := time.Parse(time.Kitchen, "12:00PM")
	tDiff := tEnd.Sub(tStart)
	// first randomly insert 10 messages timestamped between our
	// times.
	msgs := prepareMessages(c, s.db, tStart, tEnd, tDiff/10)

	var got []convo.Message
	// does GetMessageRange match the expected values?
	err := s.db.View(func(tx *bolt.Tx) (e error) {
		got, e = convo.GetMessageRange(
			"hello",
			tStart, tEnd,
			10,
			tx,
		)
		return
	})
	c.Check(err, IsNil)
	c.Check(got, DeepEquals, msgs)
}

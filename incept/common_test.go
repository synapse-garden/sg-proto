package incept_test

import (
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
	. "gopkg.in/check.v1"
)

func assertNoTickets(c *C) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		keys, vals, err := testing.FindAll(tx, incept.TicketBucket)
		c.Assert(err, IsNil)
		c.Check(len(keys) == 0, Equals, true)
		c.Check(len(vals) == 0, Equals, true)
		return nil
	}
}

func assertTicketsExist(c *C, ts ...incept.Ticket) func(*bolt.Tx) error {
	keys := make([][]byte, len(ts))
	for i, t := range ts {
		keys[i] = t.Bytes()
	}
	return func(tx *bolt.Tx) error {
		_, err := testing.FindForKeys(
			tx,
			incept.TicketBucket,
			keys...,
		)
		c.Assert(err, IsNil)
		return nil
	}
}

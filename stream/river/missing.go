package river

import (
	"fmt"
	"strconv"

	"github.com/boltdb/bolt"
	"github.com/synapse-garden/sg-proto/store"
)

// Missing is an interface which can be implemented by an error value to
// show that some IDs were missing in a survey.
type Missing interface {
	IDs() []uint64
}

// missing implements Missing on []uint64.
type missing []uint64

// IDs implements Missing.IDs on missing.
func (m missing) IDs() []uint64 { return []uint64(m) }

// Error implements error.Error on missing.
func (m missing) Error() string {
	plstr := ""
	if len(m) > 1 {
		plstr = "s"
	}

	return fmt.Sprintf("no response from ID%s: %+v", plstr, []uint64(m))
}

// CheckMissing returns a bolt View function which checks the IDs of m,
// and if any remain, returns a new Missing with the remaining IDs.
func CheckMissing(in ...store.Bucket) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		b, err := store.GetNestedBucket(
			tx.Bucket(RiverBucket),
			in...,
		)
		switch {
		case store.IsMissingBucket(err):
			return errStreamMissing(err.(store.ErrMissingBucket))
		case err != nil:
			return err
		}

		// Check the bucket for remaining IDs.
		var remain missing
		c := b.Cursor()
		for k, _ := c.First(); k != nil; k, _ = c.Next() {
			id, err := strconv.ParseUint(string(k), 10, 64)
			if err != nil {
				return err
			}
			remain = append(remain, id)
		}

		if len(remain) > 0 {
			return remain
		}
		return nil
	}
}

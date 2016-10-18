package river

import (
	"bytes"

	"github.com/synapse-garden/sg-proto/store"

	"github.com/boltdb/bolt"
)

// RiverBucket stores Rivers and their users.  Buckets in RiverBucket
// correspond to Streams from StreamBucket by ID, and every River ID in
// the bucket corresponds to a connected River.
var RiverBucket = store.Bucket("rivers")

// River is a simplified sender and receiver which can be implemented by
// mangos.Socket.
type River interface {
	Close() error
	Send([]byte) error
	Recv() ([]byte, error)
}

// CheckRiverNotExists returns an error if the given River exists.
func CheckRiverNotExists(id, streamID string) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		b := tx.Bucket(RiverBucket).Bucket([]byte(streamID))
		if b == nil {
			return nil
		}
		bid := []byte(id)
		k, _ := b.Cursor().Seek(bid)
		if !bytes.Equal(bid, k) {
			return nil
		}

		return errExists(id)
	}
}

// ClearRivers eliminates all databased Rivers.  Use this on startup.
func ClearRivers(tx *bolt.Tx) error {
	b := tx.Bucket(RiverBucket)
	return b.ForEach(func(k, v []byte) error {
		if err := b.DeleteBucket(k); err != nil {
			return err
		}
		return nil
	})
}

// DeleteRiver deletes the River from the database.  It must be used
// within a transaction where the River is also closed.
func DeleteRiver(id, streamID string) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		b := tx.Bucket(RiverBucket).Bucket([]byte(streamID))
		return b.Delete([]byte(id))
	}
}

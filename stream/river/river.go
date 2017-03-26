package river

import (
	"bytes"

	"github.com/boltdb/bolt"
)

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

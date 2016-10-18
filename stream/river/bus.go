package river

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/go-mangos/mangos"
	"github.com/go-mangos/mangos/protocol/bus"
	"github.com/go-mangos/mangos/transport/inproc"
	"github.com/pkg/errors"
)

// NewBus creates a new bus River, registers it in the DB, connects it
// to any Rivers in the streamID bucket in RiverBucket, and returns it
// or any error.  It will be created with the given string ID, which is
// its address.
func NewBus(id, streamID string, tx *bolt.Tx) (r River, e error) {
	b, err := tx.Bucket(RiverBucket).CreateBucketIfNotExists([]byte(
		streamID,
	))
	if err != nil {
		return nil, errors.Wrap(err, "failed to create bucket")
	}

	sock, err := bus.NewSocket()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create socket")
	}
	sock.AddTransport(inproc.NewTransport())

	defer func() {
		if e != nil {
			if e2 := sock.Close(); e2 != nil {
				e = errors.Wrapf(e,
					"error while closing River "+
						"after error: %s",
					e2.Error(),
				)
			}
		}
	}()

	c := b.Cursor()
	var clients [][]byte
	for k, _ := c.First(); ; k, _ = c.Next() {
		if k == nil {
			if err = b.Put([]byte(id), nil); err != nil {
				return nil, errors.Wrap(err,
					"failed to write river to DB")
			}
			err = sock.Listen(fmt.Sprintf(
				"inproc://%s/%s", streamID, id,
			))
			switch {
			case err == mangos.ErrAddrInUse:
				return nil, errExists(id)
			case err != nil:
				return nil, errors.Wrap(err,
					"failed to start listening")
			}
			break
		}

		clients = append(clients, k)
	}

	for _, client := range clients {
		err := sock.Dial(fmt.Sprintf(
			"inproc://%s/%s", streamID, client,
		))
		if err != nil {
			return nil, errors.Wrapf(err,
				"failed to dial client %#q", client)
		}
	}

	return sock, nil
}

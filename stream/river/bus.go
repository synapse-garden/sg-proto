package river

import (
	"fmt"
	"strconv"

	"github.com/synapse-garden/sg-proto/store"

	"github.com/boltdb/bolt"
	"github.com/go-mangos/mangos"
	mg_bus "github.com/go-mangos/mangos/protocol/bus"
	"github.com/go-mangos/mangos/transport/inproc"
	"github.com/pkg/errors"
)

// Bus is a River implemented by the mangos BUS protocol.
type Bus interface {
	River
	ID() uint64
}

type bus struct {
	mangos.Socket
	id uint64
}

func (b bus) ID() uint64 { return b.id }

// NewBus creates a new Bus River, registers it in the DB, connects it
// to any Rivers in the streamID bucket in RiverBucket, and returns it
// or any error.  It will be created with the given string ID, which is
// its address.
//
// Buses are created in /rivers/{streamID}/{id}/ bucket sequentially.
func NewBus(id, streamID string, tx *bolt.Tx) (r Bus, e error) {
	strB, err := store.MakeNestedBucket(
		tx.Bucket(RiverBucket),
		[]byte(streamID),
	)
	if err != nil {
		return nil, errors.Wrap(err,
			"failed to create stream bucket")
	}

	var clients []string
	bucketPrefix := "inproc://" + streamID + "/"
	// For each bucket in the Stream bucket, read all contained ids.
	c := strB.Cursor()
	for out, _ := c.First(); out != nil; out, _ = c.Next() {
		prefix := bucketPrefix + string(append(out, '/'))
		cIn := strB.Bucket(out).Cursor()
		for in, _ := cIn.First(); in != nil; in, _ = cIn.Next() {
			// ATTENTION:
			//   Note that converting from []byte to string
			//   causes a copy.  This is necessary because
			//   the internal representation in Bolt can
			//   change.
			clients = append(clients, prefix+string(in))
		}
	}
	if err != nil {
		return nil, errors.Wrap(err, "failed to read clients")
	}

	idB, err := strB.CreateBucketIfNotExists([]byte(id))
	if err != nil {
		return nil, errors.Wrap(err,
			"failed to create id bucket")
	}

	sock, err := mg_bus.NewSocket()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create socket")
	}
	sock.AddTransport(inproc.NewTransport())

	seq, err := idB.NextSequence()
	if err != nil {
		return nil, errors.Wrap(err, "failed to get sequence")
	}

	uintStr := strconv.FormatUint(seq, 10)
	if err = idB.Put([]byte(uintStr), nil); err != nil {
		return nil, errors.Wrap(err,
			"failed to write river to DB")
	}

	err = sock.Listen(fmt.Sprintf(
		"inproc://%s/%s/%s",
		streamID,
		id,
		uintStr,
	))
	if err != nil {
		return nil, errors.Wrap(err,
			"failed to start listening")
	}

	for _, client := range clients {
		if err := sock.Dial(client); err != nil {
			if e2 := sock.Close(); e2 != nil {
				return nil, errors.Wrapf(e2,
					"error while closing River "+
						"after error: %s",
					err.Error())
			}
			return nil, errors.Wrapf(err,
				"failed to dial client %#q",
				client,
			)
		}
	}

	return bus{Socket: sock, id: seq}, nil
}

// DeleteBus deletes the Bus River for the given streamID and id from
// the database.  It should be used within a transaction where the
// Bus River is also closed.
func DeleteBus(id, streamID string, seq uint64) func(*bolt.Tx) error {
	sID, bID := []byte(streamID), []byte(id)
	return func(tx *bolt.Tx) error {
		b, err := store.GetNestedBucket(
			tx.Bucket(RiverBucket),
			sID, bID,
		)
		switch {
		case store.IsMissingBucket(err):
			return errStreamMissing(err.(store.ErrMissingBucket))
		case err != nil:
			return err
		}

		seqBs := []byte(strconv.FormatUint(seq, 10))
		return b.Delete(seqBs)
	}
}

package river

import (
	"bytes"

	"github.com/boltdb/bolt"
	"github.com/go-mangos/mangos"
	"github.com/go-mangos/mangos/protocol/pub"
	"github.com/go-mangos/mangos/transport/inproc"
	"github.com/pkg/errors"
)

// Pub is a Publisher River which can only Send.  To send on a Topic,
// prefix the Send with the desired Topic code and Prefix byte.
// Only Subs which are created on the given Topic, connected to the
// PubRiver, will receive messages on that Topic.
//
// NOTE: The go-mangos implementation of Pub (returned by NewPub) sends
//       messages to all connected SubRivers, but they are filtered
//       before Recv.  This is not a technique which guarantees
//       unauthorized hosts will not receive the sent bytes; they will.
//       But their Sub Recv method will not behave as though it received
//       the message.
type Pub interface {
	Close() error
	Send([]byte) error
}

// NewPub creates an inproc publisher River in the given Stream bucket
// in RiverBucket, with address id.
func NewPub(id, streamID string, tx *bolt.Tx) (r Pub, e error) {
	b, err := tx.Bucket(RiverBucket).CreateBucketIfNotExists([]byte(
		streamID,
	))
	if err != nil {
		return nil, err
	}

	sock, err := pub.NewSocket()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create socket")
	}

	sock.AddTransport(inproc.NewTransport())

	bID := []byte(id)
	if k, _ := b.Cursor().Seek(bID); bytes.Equal(k, bID) {
		return nil, errExists(id)
	}

	if err = b.Put(bID, nil); err != nil {
		return nil, errors.Wrap(err,
			"failed to write river to DB")
	}
	err = sock.Listen("inproc://" + streamID + "/" + id)
	switch {
	case err == mangos.ErrAddrInUse:
		return nil, errExists(id)
	case err != nil:
		return nil, errors.Wrap(err,
			"failed to start listening")
	}

	return sock, nil
}

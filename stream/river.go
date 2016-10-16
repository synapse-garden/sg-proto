package stream

import (
	"bytes"
	"fmt"

	"github.com/synapse-garden/sg-proto/store"

	"github.com/boltdb/bolt"
	"github.com/go-mangos/mangos"
	"github.com/go-mangos/mangos/protocol/bus"
	"github.com/go-mangos/mangos/protocol/pub"
	mg_sub "github.com/go-mangos/mangos/protocol/sub"
	"github.com/go-mangos/mangos/transport/inproc"
	"github.com/pkg/errors"
)

// RiverBucket stores Rivers and their users.  Buckets in RiverBucket
// correspond to Streams from StreamBucket by ID, and every River ID in
// the bucket corresponds to a connected River.
var RiverBucket = store.Bucket("rivers")

type errRiverExists string

// Error implements error on errRiverExists.
func (e errRiverExists) Error() string {
	return fmt.Sprintf("river %#q already exists", string(e))
}

// River is a simplified sender and receiver which can be implemented by
// mangos.Socket.
type River interface {
	Close() error
	Send([]byte) error
	Recv() ([]byte, error)
}

// IsRiverExists returns true if the error is an existent River when one
// is not expected.
func IsRiverExists(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(errRiverExists)
	return ok
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

		return errRiverExists(id)
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

// PubRiver is a Publisher River which can only Send.  To send on a
// Topic, prefix the Send with the desired Topic code and Prefix byte.
// Only SubRivers which are created on the given Topic, connected to the
// PubRiver, will receive messages on that Topic.
//
// NOTE: The go-mangos implementation of PubRiver (returned by NewPub)
//       sends messages to all connected SubRivers, but they are
//       filtered before Recv.  This is not a technique which guarantees
//       unauthorized hosts will not receive the sent bytes; they will.
//       But their SubRiver Recv method will not behave as though it
//       received the message.
type PubRiver interface {
	Close() error
	Send([]byte) error
}

// NewPub creates an inproc publisher River in the given Stream bucket
// in RiverBucket, with address id.
func NewPub(id, streamID string, tx *bolt.Tx) (r PubRiver, e error) {
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

	defer func() {
		if e != nil {
			if e2 := sock.Close(); e2 != nil {
				e = errors.Wrapf(e,
					"error while closing River "+
						"after error: %s",
					e.Error())
			}
		}
	}()

	if b.Get([]byte(id)) != nil {
		return nil, errRiverExists(id)
	}

	if err = b.Put([]byte(id), nil); err != nil {
		return nil, errors.Wrap(err,
			"failed to write river to DB")
	}
	err = sock.Listen("inproc://" + streamID + "/" + id)
	switch {
	case err == mangos.ErrAddrInUse:
		return nil, errRiverExists(id)
	case err != nil:
		return nil, errors.Wrap(err,
			"failed to start listening")
	}

	return sock, nil
}

// SubRiver is a Subscriber River which can only Recv.  It can be
// implemented on a byte prefix Topic, which is to be removed before
// Recv returns.
type SubRiver interface {
	Close() error
	Recv() ([]byte, error)
}

type sub struct {
	SubRiver
	// TODO: trie-based code length match
	topics map[byte]Topic
}

func (s sub) Recv() ([]byte, error) {
	bs, err := s.SubRiver.Recv()
	if err != nil {
		return nil, err
	}

	// Note that sub is always made in NewSub so its length should
	// never be 0.
	if topic, ok := s.topics[bs[0]]; ok {
		return bs[topic.Len():], nil
	}

	return nil, fmt.Errorf("received unknown prefix %#v", bs[0])
}

// NewSub creates an inproc subscriber River which connects to all
// publisher Rivers in the given streamID bucket in RiverBucket,
// subscribing on the given Topics, or all topics if no Topic is given.
func NewSub(
	streamID string,
	tx *bolt.Tx,
	topics ...Topic,
) (r SubRiver, e error) {
	b := tx.Bucket(RiverBucket).Bucket([]byte(streamID))
	if b == nil {
		return nil, errStreamMissing(streamID)
	}

	sock, err := mg_sub.NewSocket()
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
					e.Error())
			}
		}
	}()

	c := b.Cursor()
	var servers [][]byte
	for k, _ := c.First(); k != nil; k, _ = c.Next() {
		servers = append(servers, k)
	}

	if len(topics) == 0 {
		topics = []Topic{Global}
	}

	topicMap := make(map[byte]Topic)
	for _, t := range topics {
		pre := t.Prefix()
		if top, ok := topicMap[pre]; ok {
			return nil, errors.Errorf(
				"redundant topics %#q, %#q",
				top.Name(), t.Name(),
			)
		}
		topicMap[pre] = t
		err = sock.SetOption(mangos.OptionSubscribe, t.Code())
		if err != nil {
			return nil, errors.Wrapf(err,
				"bad topic %#q", t.Name())
		}
	}

	for _, server := range servers {
		err = sock.Dial(fmt.Sprintf(
			"inproc://%s/%s", streamID, server,
		))
		if err != nil {
			return nil, errors.Wrapf(err,
				"failed to dial server %#q",
				server)
		}
	}

	return sub{
		SubRiver: sock,
		topics:   topicMap,
	}, nil
}

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
				return nil, errRiverExists(id)
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

// DeleteRiver deletes the River from the database.  It must be used
// within a transaction where the River is also closed.
func DeleteRiver(id, streamID string) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		b := tx.Bucket(RiverBucket).Bucket([]byte(streamID))
		return b.Delete([]byte(id))
	}
}

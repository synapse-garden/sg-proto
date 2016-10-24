package river

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/go-mangos/mangos"
	mg_sub "github.com/go-mangos/mangos/protocol/sub"
	"github.com/go-mangos/mangos/transport/inproc"
	"github.com/pkg/errors"
)

// Sub is a Subscriber River which can only Recv.  It can be implemented
// on a byte prefix Topic, which is to be removed before Recv returns.
type Sub interface {
	Close() error
	Recv() ([]byte, error)
}

type sub struct {
	Sub
	// TODO: trie-based code length match
	topics map[byte]Topic
}

func (s sub) Recv() ([]byte, error) {
	bs, err := s.Sub.Recv()
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
) (r Sub, e error) {
	b := tx.Bucket(RiverBucket).Bucket([]byte(streamID))
	if b == nil {
		return nil, errStreamMissing(streamID)
	}

	sock, err := mg_sub.NewSocket()
	if err != nil {
		return nil, errors.Wrap(err, "failed to create socket")
	}

	sock.AddTransport(inproc.NewTransport())

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
			if e2 := sock.Close(); e2 != nil {
				return nil, errors.Wrapf(e2,
					"error while closing River "+
						"after error: %s",
					err.Error(),
				)
			}
			return nil, errors.Wrapf(err,
				"failed to dial server %#q",
				server)
		}
	}

	return sub{
		Sub:    sock,
		topics: topicMap,
	}, nil
}

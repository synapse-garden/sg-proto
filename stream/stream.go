package stream

import (
	"encoding/json"

	"github.com/synapse-garden/sg-proto/store"

	"github.com/boltdb/bolt"
)

// Needed interfaces:
//  - Storer: io.Writer which writes objects as bytes down to disk,
//            doesn't need to be threadsafe, just keep it running
//  - Router: "sandwich" between Storer and Coupler, forwards messages
//            from client to client(s); Storer can be a client
//  - Coupler: safely couples and decouples clients with routers
//  - Subscriber: a client for a Router
//  - Subset: a function on a stream that causes only a subset of stream
//    messages to be received by subscribers (?)
//  - Stream: the object representing the user permissions on a stream

// Package API 0.1.0:
//  - Stream is an abstract handle to some flow of data with owner,
//    topics, readers, and writers
//  - Someone who has a valid Stream handle can Subscribe to a Stream
//  - There can be public streams which clients can join and leave
//    without mutating
//  - Some user wants to create a stream.
//    > Create(s *Stream) func(*bolt.Tx) error
//      + Fails if exists
//      + Fails if invalid (e.g. read / write user etc not exist)
//      + Returns valid Stream if exists.
//  - !!!! Some user wants to get a Stream which is a subset of messages
//    on another Stream.
//  - Some user wants to mutate a stream.
//    > Update(s *Stream) func(*bolt.Tx) error
//  - Some user wants to destroy a stream.
//    > Delete(s *Stream) func(*bolt.Tx) error
//  - Some user wants to join or leave a stream.
//    > Subscribe(s *Stream, c *ws.Conn) func(*bolt.Tx) error
//    > Unsubscribe(s *Stream) func(*bolt.Tx) error
//  - Some user wants to use notifications
//    > Separate notifs package which maps directly to streams
//  - Some user wants to be notified when their membership in a stream
//    changes.
//  - Some user wants to connect to some stream.
//  - Some user wants to load archived messages from a stream.
//
// Package API 0.0.1:
//  - Ephemeral streams (?)
//  - Get(s *Stream) func(*bolt.Tx) error
//  - Find(userID string, {stream id => *stream}) func(*bolt.Tx) error
//    > map gets populated with matches
//  - Create(s *Stream) func(*bolt.Tx) error
//    > Rejects if stream exists
//  - Join(s *Stream, c *ws.Conn) func(*bolt.Tx) error
//    > Rejects if no permission
//    > If router not yet created, spawns router and attaches client
//    > May need to rethink API
//  - Update(s *Stream) func(*bolt.Tx) error
//    > E.g. add user, hangs up users if removed (how?)
//  - Delete(s *Stream) func(*bolt.Tx) error
//    > Hangs up all users
//
//  - Some kind of internal API between the router etc and the CRUD
//    Stream object (River?)
//  - Users can hang up without error
//  - Clients are tracked and can be removed (by other packages? River?)

// Buckets.  Note that each River in RiverBucket is a bucket.
var (
	StreamBucket = store.Bucket("streams")
)

// Stream represents user access to an underlying Router, Coupler, etc.
type Stream struct {
	//
	ID string `json:"id"`

	Owner string `json:"owner"`

	Name string `json:"name"`

	Readers map[string]bool `json:"readers,omitempty"`
	Writers map[string]bool `json:"writers,omitempty"`
}

// Resource implements store.Resourcer on Stream.
func (Stream) Resource() store.Resource { return "streams" }

// CheckNotExist returns a function which returns nil if the Stream with
// the given ID does not exist.
func CheckNotExist(id string) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		err := store.CheckNotExist(StreamBucket, []byte(id))(tx)
		if store.IsExists(err) {
			return errExists(id)
		}
		return err
	}
}

// CheckExists returns a function which returns nil if the Stream with
// the given ID exists.
func CheckExists(id string) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		err := store.CheckExists(StreamBucket, []byte(id))(tx)
		if store.IsMissing(err) {
			return errMissing(id)
		}
		return err
	}
}

// Get returns a function which loads the stream for the given ID, or
// returns any error.
func Get(s *Stream, id string) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		err := store.Unmarshal(StreamBucket, s, []byte(id))(tx)
		switch {
		case store.IsMissing(err):
			return errMissing(id)
		case err != nil:
			return err
		}
		return nil
	}
}

// GetAll returns a function which unmarshals all streams for which the
// user has ownership.  If Filters are passed, only streams for which
// filter.Member(stream) == true will be returned.
func GetAll(user string, filters ...Filter) func(*bolt.Tx) ([]*Stream, error) {
	var result []*Stream

	defaultFilter := MultiOr{
		ByOwner(user),
		ByReader(user),
		ByWriter(user),
	}
	otherFilters := MultiAnd(filters)

	return func(tx *bolt.Tx) ([]*Stream, error) {
		b := tx.Bucket(StreamBucket)
		// TODO: channel producer / consumer to speed this up
		// TODO: Other ways to improve this so users aren't
		//       constantly hammering the database
		// TODO: Benchmark test
		err := b.ForEach(func(k, v []byte) error {
			next := new(Stream)
			if err := json.Unmarshal(v, next); err != nil {
				return err
			}

			switch {
			case !defaultFilter.Member(next):
				return nil
			case !otherFilters.Member(next):
				return nil
			}

			result = append(result, next)
			return nil
		})

		return result, err
	}
}

// Upsert returns a function which inserts or updates the given Stream,
// or returns any error.
func Upsert(s *Stream) func(*bolt.Tx) error {
	return store.Marshal(StreamBucket, s, []byte(s.ID))
}

// Delete deletes the stream with the given ID.
func Delete(id string) func(*bolt.Tx) error {
	return store.Delete(StreamBucket, []byte(id))
}

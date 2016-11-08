package convo

import (
	"encoding/json"

	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream"

	"github.com/boltdb/bolt"
)

// Bucket constants.
var (
	ConvoBucket   = store.Bucket("convos")
	ScribeBucket  = store.Bucket("scribes")
	MessageBucket = store.Bucket("messages")
)

// Convo is an alias for stream.Stream, i.e., it is a struct for
// controlling access and ownership of a websocket based connection.
type Convo stream.Stream

// CheckNotExist returns a function which returns nil if the Convo with
// the given ID does not exist.
func CheckNotExist(id string) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		err := store.CheckNotExist(ConvoBucket, []byte(id))(tx)
		if store.IsExists(err) {
			return errExists(id)
		}
		return err
	}
}

// CheckExists returns a function which returns nil if the Convo with
// the given ID exists.
func CheckExists(id string) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		err := store.CheckExists(ConvoBucket, []byte(id))(tx)
		if store.IsMissing(err) {
			return errMissing(id)
		}
		return err
	}
}

// Get returns a function which loads the convo for the given ID, or
// returns any error.
func Get(c *Convo, id string) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		err := store.Unmarshal(ConvoBucket, c, []byte(id))(tx)
		switch {
		case store.IsMissing(err):
			return errMissing(id)
		case err != nil:
			return err
		}
		return nil
	}
}

// Upsert inserts or updates a Convo in the database.  It should already
// have an ID set using something like uuid.NewV4.
func Upsert(c *Convo) func(tx *bolt.Tx) error {
	return store.Marshal(ConvoBucket, c, []byte(c.ID))
}

// Delete deletes the convo with the given ID.
func Delete(id []byte) func(tx *bolt.Tx) error {
	return store.Delete(ConvoBucket, id)
}

// GetAll returns a function which unmarshals all convos for which the
// user has ownership.  If Filters are passed, only convos for which
// filter.Member(convo) == true will be returned.
func GetAll(
	user string,
	filters ...stream.Filter,
) func(*bolt.Tx) ([]*Convo, error) {
	var result []*Convo

	defaultFilter := stream.MultiOr{
		stream.ByOwner(user),
		stream.ByReader(user),
		stream.ByWriter(user),
	}
	otherFilters := stream.MultiAnd(filters)

	return func(tx *bolt.Tx) ([]*Convo, error) {
		b := tx.Bucket(ConvoBucket)
		// TODO: channel producer / consumer to speed this up
		// TODO: Other ways to improve this so users aren't
		//       constantly hammering the database
		// TODO: Benchmark test
		// TODO: Eliminate redundant code between Stream / Convo.
		err := b.ForEach(func(k, v []byte) error {
			next := new(Convo)
			if err := json.Unmarshal(v, next); err != nil {
				return err
			}
			strConv := (*stream.Stream)(next)
			switch {
			case !defaultFilter.Member(strConv):
				return nil
			case !otherFilters.Member(strConv):
				return nil
			}

			result = append(result, next)
			return nil
		})

		return result, err
	}
}

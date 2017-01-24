package text

import (
	"github.com/synapse-garden/sg-proto/store"

	"github.com/boltdb/bolt"
)

// TextBucket is the database location where text is stored.
var TextBucket = store.Bucket("text")

// ID is a reference to a text object in the database.
type ID store.ID

// MakeID makes an ID from the given string by hashing with the given
// store.ID.
func MakeID(i store.ID, from string) ID {
	return ID(i.HashWith(from))
}

// Store implements store.Storer on ID.
func (i ID) Store(what interface{}) func(*bolt.Tx) error {
	return store.Marshal(TextBucket, what, i[:])
}

// Load implements store.Loader on ID.
func (i ID) Load(into interface{}) func(*bolt.Tx) error {
	if tStr, ok := into.(*string); ok {
		return store.Unmarshal(TextBucket, tStr, i[:])
	}

	return store.Errorf("unexpected Load argument of type %T", into)
}

// Delete implements store.Deleter on ID.
func (i ID) Delete(tx *bolt.Tx) error {
	return tx.Bucket(TextBucket).Delete(i[:])
}

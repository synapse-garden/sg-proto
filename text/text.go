package text

import (
	"github.com/synapse-garden/sg-proto/store"

	"github.com/boltdb/bolt"
	uuid "github.com/satori/go.uuid"
)

// TextBucket is the database location where text is stored.
var TextBucket = store.Bucket("text")

// Text is a reference to a text object in the database.
type Text uuid.UUID

// Store implements store.Storer on Text.
func (t Text) Store(what interface{}) func(tx *bolt.Tx) error {
	return store.Marshal(TextBucket, what, t[:])
}

// Load implements store.Loader on Text.
func (t Text) Load(into interface{}) func(tx *bolt.Tx) error {
	if tStr, ok := into.(*string); ok {
		return store.Unmarshal(TextBucket, tStr, t[:])
	}

	return store.Errorf("unexpected Load argument of type %T", into)
}

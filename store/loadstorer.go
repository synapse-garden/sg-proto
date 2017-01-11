package store

import "github.com/boltdb/bolt"

// Loader is a reference to an entity in the DB which can be Loaded into
// the given argument.  This would typically be implemented using an ID.
// Consider it equivalent to a reference type, where the "address" is a
// databased entity.  The *bolt.Tx may be a Read or a Write transaction.
type Loader interface {
	Load(interface{}) func(tx *bolt.Tx) error
}

// Storer is an entity which can store its representation using a bolt
// Write transaction.  Typically this should be implemented by an ID.  A
// reference type should be passed so that Store can set its ID.
type Storer interface {
	Store(interface{}) func(tx *bolt.Tx) error
}

// LoadStorer is composed of a Loader and a Storer.
type LoadStorer interface {
	Loader
	Storer
}

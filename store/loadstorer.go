package store

import (
	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
)

// Loader is a reference to an entity in the DB which can be Loaded into
// the given argument.  This would typically be implemented using an ID.
// Consider it equivalent to a reference type, where the "address" is a
// databased entity.  The *bolt.Tx may be a Read or a Write transaction.
type Loader interface {
	Load(interface{}) func(*bolt.Tx) error
}

type Loaders []Loader

// Storer is an entity which can store its representation using a bolt
// Write transaction.  Typically this should be implemented by an ID.  A
// reference type should be passed so that Store can set its ID.
type Storer interface {
	Store(interface{}) func(*bolt.Tx) error
}

type Storers []Storer

type Deleter interface {
	Delete(*bolt.Tx) error
}

type Deleters []Deleter

// LoadStorer is composed of a Loader and a Storer.
type LoadStorer interface {
	Loader
	Storer
}

type LoadStoreDeleter interface {
	Loader
	Storer
	Deleter
}

// StoreAll stores all the given objects.
func (ss Storers) StoreAll(what ...interface{}) func(*bolt.Tx) error {
	lenS := len(ss)
	lenW := len(what)

	return func(tx *bolt.Tx) error {
		if lenS != lenW {
			return errors.Errorf("number of storers (%d) "+
				"must match number of objects (%d)",
				lenS, lenW,
			)
		}

		for i, s := range ss {
			if err := s.Store(what[i])(tx); err != nil {
				return err
			}
		}

		return nil
	}
}

// LoadAll loads all the given objects.
func (ls Loaders) LoadAll(into ...interface{}) func(*bolt.Tx) error {
	lenL := len(ls)
	lenI := len(into)

	return func(tx *bolt.Tx) error {
		if lenL != lenI {
			return errors.Errorf("number of loaders (%d) "+
				"must match number of objects (%d)",
				lenL, lenI,
			)
		}

		for i, l := range ls {
			if err := l.Load(into[i])(tx); err != nil {
				return err
			}
		}

		return nil
	}
}

func (ds Deleters) DeleteAll(tx *bolt.Tx) error {
	for _, d := range ds {
		if err := d.Delete(tx); err != nil {
			return err
		}
	}

	return nil
}

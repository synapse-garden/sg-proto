package store

import (
	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
)

type Version string

const (
	VerNone       = Version("")
	VerAlpha001_2 = Version("0.0.1-alpha-2")

	VerCurrent = VerAlpha001_2
)

var (
	VersionBucket = []byte("version")

	migrations = map[Version]map[Version]func(*bolt.Tx) error{
		VerNone: {VerAlpha001_2: PutV(VerCurrent)},
	}
)

// Bucket is an identifier for a package constant to define the BoltDB
// bucket where a resource is stored.
//
// TODO: nested Buckets?
type Bucket []byte

func Prep(buckets ...Bucket) func(*bolt.Tx) error {
	return Wrap(
		Migrate(VerCurrent),
		SetupBuckets(buckets...),
	)
}

func SetupBuckets(buckets ...Bucket) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		for _, bucket := range buckets {
			_, err := tx.CreateBucketIfNotExists(bucket)
			if err != nil {
				return err
			}
		}

		return nil
	}
}

func Migrate(v Version) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		b := tx.Bucket(VersionBucket)
		if b != nil {
			oldVer := Version(b.Get([]byte("version")))
			if v != oldVer {
				return MigrateFrom(tx, oldVer, v)
			}
			return nil
		}
		return MigrateFrom(tx, VerNone, v)
	}
}

func MigrateFrom(tx *bolt.Tx, from, to Version) error {
	msFrom, ok := migrations[from]
	if !ok {
		return errors.Errorf("no migration defined from version %#q", from)
	}
	if from == to {
		return nil
	}
	mTo, ok := msFrom[to]
	if !ok {
		return errors.Errorf("no migration defined from version %#q to %#q", from, to)
	}
	return mTo(tx)
}

func PutV(v Version) func(*bolt.Tx) error {
	return Wrap(
		func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists(VersionBucket)
			return err
		},
		Put(VersionBucket, []byte("version"), []byte(v)),
	)
}

func Wrap(apps ...func(*bolt.Tx) error) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		for _, app := range apps {
			if err := app(tx); err != nil {
				return err
			}
		}
		return nil
	}
}

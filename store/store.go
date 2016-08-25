package store

import (
	"bytes"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
)

var (
	Version       = []byte("0.0.1-alpha-2")
	VersionBucket = []byte("version")
)

type Bucket []byte

func Prep(buckets ...Bucket) func(*bolt.Tx) error {
	return Wrap(
		Migrate(Version),
		SetupBuckets(buckets...),
		Put(VersionBucket, []byte("version"), Version),
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

func Migrate(version []byte) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		if b := tx.Bucket(VersionBucket); b != nil {
			newVer := b.Get([]byte("version"))
			if bytes.Compare(version, newVer) != 0 {
				return MigrateFrom(tx, version)
			}
		}
		return nil
	}
}

func MigrateFrom(tx *bolt.Tx, from []byte) error {
	if bytes.Compare(from, Version) == 0 {
		// No need to migrate, already on latest version
		return nil
	}
	migration, ok := map[string]func(*bolt.Tx) error{
		"0.0.1-alpha-2": Wrap(),
	}[string(from)]
	if !ok {
		return errors.Errorf("no migration defined from version %#q", from)
	}
	return migration(tx)
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

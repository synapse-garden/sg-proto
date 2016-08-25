package testing

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/synapse-garden/sg-proto/store"
)

// TempDB returns a temporary DB and its temporary directory.
func TempDB(name string) (*bolt.DB, string, error) {
	d, err := ioutil.TempDir("", "sg-test")
	if err != nil {
		return nil, "", err
	}
	db, err := bolt.Open(
		filepath.Join(d, "test.db"),
		0600,
		nil,
	)
	if err != nil {
		return nil, "", err
	}

	return db, d, nil
}

func CleanupDB(db *bolt.DB) error {
	path := db.Path()
	if err := db.Close(); err != nil {
		return err
	}
	if err := os.Remove(path); err != nil {
		return err
	}
	return nil
}

func FindForKeys(tx *bolt.Tx, bucket []byte, keys ...[]byte) ([][]byte, error) {
	b := tx.Bucket(bucket)
	if b == nil {
		return nil, errors.Errorf("no such bucket %#q", bucket)
	}
	var result [][]byte
	for _, k := range keys {
		if bs := b.Get(k); bs != nil {
			result = append(result, bs)
		} else {
			return nil, store.MissingError(k)
		}
	}
	return result, nil
}

// FindAll returns a copy of all keys and values in the given bucket.
func FindAll(tx *bolt.Tx, bucket []byte) ([][]byte, [][]byte, error) {
	b := tx.Bucket(bucket)
	if b == nil {
		return nil, nil, errors.Errorf("no such bucket %#q", bucket)
	}

	c := b.Cursor()

	var resultKeys, resultValues [][]byte
	for k, v := c.First(); k != nil; k, v = c.Next() {
		resultKeys = append(resultKeys, k[:])
		resultValues = append(resultValues, v[:])
	}

	return resultKeys, resultValues, nil
}

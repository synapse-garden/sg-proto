package testing

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
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

// FindAll returns a copy of all keys and values in the given bucket.
func FindAll(tx *bolt.Tx, bucket []byte) ([][]byte, [][]byte, error) {
	b := tx.Bucket(bucket)
	if b == nil {
		return nil, nil, errors.Errorf("no such bucket %#q", bucket)
	}

	// Create a cursor for iteration.
	c := b.Cursor()

	// Iterate over items in sorted key order. This starts from the
	// first key/value pair and updates the k/v variables to the
	// next key/value on each iteration.
	//
	// The loop finishes at the end of the cursor when a nil key is returned.
	var resultKeys, resultValues [][]byte
	for k, v := c.First(); k != nil; k, v = c.Next() {
		resultKeys = append(resultKeys, k[:])
		resultValues = append(resultValues, v[:])
	}

	return resultKeys, resultValues, nil
}

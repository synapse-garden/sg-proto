package testing

import (
	"io/ioutil"
	"os"
	"path/filepath"

	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/store"

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

func Cleanup(db *bolt.DB, ccs ...rest.Cleanup) error {
	if err := rest.Cleanups(ccs).Cleanup(); err != nil {
		return errors.Wrap(err, "failed to clean up APIs")
	}

	path := db.Path()
	if err := db.Close(); err != nil {
		return errors.Wrap(err, "failed to close DB")
	}

	if err := os.Remove(path); err != nil {
		return errors.Wrapf(err,
			"failed to remove DB file %s", path,
		)
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
			return nil, &store.MissingError{
				Key:    k,
				Bucket: bucket,
			}
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

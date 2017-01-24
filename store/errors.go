package store

import (
	"fmt"

	"github.com/boltdb/bolt"
)

type ErrMissingBucket []byte

func (e ErrMissingBucket) Error() string {
	return fmt.Sprintf("no such bucket %#q", string(e))
}

func IsMissingBucket(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(ErrMissingBucket)
	return ok
}

func (m Mutation) OrMissing(tx *bolt.Tx) error {
	err := m(tx)
	if IsMissingBucket(err) || IsMissing(err) {
		return nil
	}
	return err
}

// OrMissing is a View method which ignores any Missing errors.  All
// other error values are passed through.
func (v View) OrMissing(tx *bolt.Tx) error {
	err := v(tx)
	if IsMissingBucket(err) || IsMissing(err) {
		return nil
	}
	return err
}

type KeyError struct {
	Key, Bucket []byte
}

type ExistsError KeyError

func (e ExistsError) Error() string {
	return fmt.Sprintf("key %#q already exists in bucket %#q", e.Key, e.Bucket)
}

func IsExists(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*ExistsError)
	return ok
}

type MissingError KeyError

func (m MissingError) Error() string {
	return fmt.Sprintf("key %#q not in bucket %#q", m.Key, m.Bucket)
}

func IsMissing(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*MissingError)
	return ok
}

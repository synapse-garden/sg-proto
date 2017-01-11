package store

import (
	"encoding/json"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
)

func Put(b Bucket, key, val []byte) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		return tx.Bucket(b).Put(key, val)
	}
}

func Get(b Bucket, key []byte) func(*bolt.Tx) ([]byte, error) {
	return func(tx *bolt.Tx) ([]byte, error) {
		result := tx.Bucket(b).Get(key)
		switch {
		case result == nil:
			return nil, &MissingError{
				Key:    key,
				Bucket: b,
			}
		case len(result) == 0:
			return []byte{}, nil
		}
		return result[:], nil
	}
}

func Error(err error) func(*bolt.Tx) error {
	return func(*bolt.Tx) error { return err }
}

func Errorf(fmt string, vs ...interface{}) func(*bolt.Tx) error {
	return func(*bolt.Tx) error {
		return errors.Errorf(fmt, vs...)
	}
}

func Delete(b Bucket, key []byte) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		return tx.Bucket(b).Delete(key)
	}
}

func Marshal(b Bucket, from interface{}, key []byte) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		bs, err := json.Marshal(from)
		if err != nil {
			return err
		}
		return Put(b, key, bs)(tx)
	}
}

func Unmarshal(b Bucket, to interface{}, key []byte) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		if bs := tx.Bucket(b).Get(key); bs != nil {
			return json.Unmarshal(bs, to)
		}
		return &MissingError{
			Key:    key,
			Bucket: b,
		}
	}
}

func CheckExists(b Bucket, key []byte) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		if tx.Bucket(b).Get(key) == nil {
			return &MissingError{
				Key:    key,
				Bucket: b,
			}
		}
		return nil
	}
}

func CheckNotExist(b Bucket, key []byte) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		if tx.Bucket(b).Get(key) != nil {
			return &ExistsError{
				Key:    key,
				Bucket: b,
			}
		}
		return nil
	}
}

// GetNestedBucket attempts to get the nested Bucket from the given
// Bucket, returning an error if it is missing.
func GetNestedBucket(b *bolt.Bucket, buckets ...Bucket) (*bolt.Bucket, error) {
	result := b
	for _, bk := range buckets {
		if result = result.Bucket(bk); result == nil {
			return nil, ErrMissingBucket(bk)
		}
	}
	return result, nil
}

// MakeNestedBucket creates the given Buckets on the given Bucket if
// they do not exist, returning any error, or the innermost nested
// Bucket.
func MakeNestedBucket(b *bolt.Bucket, buckets ...Bucket) (*bolt.Bucket, error) {
	result := b
	var err error
	for _, bk := range buckets {
		result, err = result.CreateBucketIfNotExists(bk)
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}

package store

import (
	"encoding/json"

	"github.com/boltdb/bolt"
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

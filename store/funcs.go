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

func Delete(b Bucket, key []byte) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		return tx.Bucket(b).Delete(key)
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

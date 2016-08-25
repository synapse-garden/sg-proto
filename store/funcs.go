package store

import "github.com/boltdb/bolt"

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

func CheckExists(b Bucket, key []byte) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		if tx.Bucket(b).Get(key) == nil {
			return MissingError(key)
		}
		return nil
	}
}

func CheckNotExist(b Bucket, key []byte) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		if tx.Bucket(b).Get(key) != nil {
			return ExistsError(key)
		}
		return nil
	}
}

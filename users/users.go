package users

import (
	"encoding/json"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/synapse-garden/sg-proto/store"
)

var UserBucket = []byte("users")

type User struct {
	Name string `json:"name,omitempty"`
	Coin int64  `json:"coin"`
}

func CheckUserExists(u *User) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		b := tx.Bucket(UserBucket)
		if val := b.Get([]byte(u.Name)); val == nil {
			return store.MissingError(u.Name)
		}
		return nil
	}
}

// Create returns a writing transaction which checks that the user has
// not yet been created, then Puts its JSON representation in UserBucket.
func Create(u *User) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		name := []byte(u.Name)
		if err := store.CheckNotExist(UserBucket, name)(tx); err != nil {
			return err
		}
		bs, err := json.Marshal(u)
		if err != nil {
			return err
		}
		return store.Put(UserBucket, name, bs)(tx)
	}
}

// ValidateNew validates a new User.
func ValidateNew(u *User) error {
	switch {
	case len(u.Name) == 0:
		return errors.New("name must not be blank")
	case u.Coin != 0:
		return errors.New("user cannot be created with coin")
	}
	return nil
}

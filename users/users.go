package users

import (
	"fmt"

	"github.com/synapse-garden/sg-proto/store"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
)

var UserBucket = []byte("users")

type User struct {
	Name string `json:"name,omitempty"`
	Coin int64  `json:"coin"`
}

type ErrExists string

func (e ErrExists) Error() string { return fmt.Sprintf("user %#q already exists", string(e)) }

type ErrMissing string

func (e ErrMissing) Error() string { return fmt.Sprintf("user %#q not found", string(e)) }

func CheckUserExists(u *User) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		err := store.CheckExists(UserBucket, []byte(u.Name))(tx)
		if store.IsMissing(err) {
			return ErrMissing(u.Name)
		}
		return err
	}
}

func CheckUserNotExist(u *User) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		err := store.CheckNotExist(UserBucket, []byte(u.Name))(tx)
		if store.IsExists(err) {
			return ErrExists(u.Name)
		}
		return err
	}
}

// Create returns a writing transaction which checks that the user has
// not yet been created, then Puts its JSON representation in UserBucket.
func Create(u *User) func(*bolt.Tx) error {
	return store.Marshal(UserBucket, u, []byte(u.Name))
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

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

func IsExists(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(ErrExists)
	return ok
}

func IsMissing(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(ErrMissing)
	return ok
}

// CheckUsersExist checks that the Users with the given names exist.
func CheckUsersExist(names ...string) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		for _, name := range names {
			err := store.CheckExists(UserBucket, []byte(name))(tx)
			if store.IsMissing(err) {
				return ErrMissing(name)
			} else if err != nil {
				return err
			}
		}

		return nil
	}
}

func CheckNotExist(names ...string) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) (e error) {
		for _, name := range names {
			e = store.CheckNotExist(UserBucket, []byte(name))(tx)
			if store.IsExists(e) {
				return ErrExists(name)
			}
		}
		return
	}
}

// Create returns a writing transaction which checks that the user has
// not yet been created, then Puts its JSON representation in UserBucket.
func Create(u *User) func(*bolt.Tx) error {
	return store.Marshal(UserBucket, u, []byte(u.Name))
}

func Delete(userID string) store.Mutation {
	return store.Delete(UserBucket, []byte(userID))
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

// AddCoin returns a store.Mutation which adds the given amount of coin
// to the given user.  It presumes the user already exists in the DB and
// has a valid Name set.  It sets the given User's Coin to the new value
// in the DB.
func AddCoin(u *User, coin int64) store.Mutation {
	nbs := []byte(u.Name)
	return func(tx *bolt.Tx) error {
		into := new(User)

		err := store.Unmarshal(UserBucket, into, nbs)(tx)
		if err != nil {
			return err
		}

		into.Coin += coin
		u.Coin = into.Coin

		return store.Marshal(UserBucket, into, nbs)(tx)
	}
}

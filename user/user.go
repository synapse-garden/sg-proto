package user

import (
	"encoding/json"
	"fmt"

	"github.com/boltdb/bolt"
)

var UserBucket = []byte("users")

type ExistsError []byte

func (e ExistsError) Error() string {
	return fmt.Sprintf("user %#q already exists", e)
}

type User struct {
	Name string
	Coin int64
}

func Create(u *User) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		b := tx.Bucket(UserBucket)
		if len(b.Get([]byte(u.Name))) != 0 {
			return ExistsError(u.Name)
		}
		if bs, err := json.Marshal(u); err != nil {
			return err
		} else {
			return b.Put([]byte(u.Name), bs)
		}
	}
}

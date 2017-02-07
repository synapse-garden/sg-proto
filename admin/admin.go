package admin

import (
	"bytes"
	"crypto/sha256"
	"fmt"

	uuid "github.com/satori/go.uuid"
	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/store"

	"github.com/boltdb/bolt"
)

var AdminBucket = store.Bucket("admin")

type ErrNotFound auth.Token

func (e ErrNotFound) Error() string {
	return fmt.Sprintf("no such admin token %#q", string(e))
}

func IsNotFound(e error) bool {
	if e == nil {
		return false
	}
	_, ok := e.(ErrNotFound)
	return ok
}

func NewToken(token auth.Token) func(*bolt.Tx) error {
	salt := uuid.NewV4()
	salted := sha256.Sum256(append(token, salt.Bytes()...))

	return store.Wrap(
		store.Put(AdminBucket, []byte("token"), salted[:]),
		store.Put(AdminBucket, []byte("salt"), salt.Bytes()),
	)
}

func CheckExists(tx *bolt.Tx) error {
	err := store.CheckExists(AdminBucket, []byte("token"))(tx)
	if store.IsMissing(err) {
		return ErrNotFound([]byte(""))
	}
	return err
}

func CheckToken(token auth.Token) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		var salt []byte
		salt, err := store.Get(AdminBucket, []byte("salt"))(tx)
		if store.IsMissing(err) {
			return ErrNotFound(token)
		} else if err != nil {
			return err
		}
		adminToken, err := store.Get(AdminBucket, []byte("token"))(tx)
		if store.IsMissing(err) {
			return ErrNotFound(token)
		} else if err != nil {
			return err
		}

		salted := sha256.Sum256(append(token, salt...))
		if !bytes.Equal(adminToken, salted[:]) {
			return ErrNotFound(token)
		}
		return nil
	}
}

package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"

	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	"github.com/satori/go.uuid"
)

// LoginBucket is a bucket for logins.
var LoginBucket = store.Bucket("logins")

// Login is a record of a User authentication.  It is a User with a PWHash.
type Login struct {
	users.User
	PWHash []byte    `json:"pwhash"`
	Salt   uuid.UUID `json:"salt"`
}

type ErrExists string

func (e ErrExists) Error() string { return fmt.Sprintf("login for user %#q already exists", string(e)) }

type ErrMissing string

func (e ErrMissing) Error() string { return fmt.Sprintf("login for user %#q not found", string(e)) }

type ErrInvalid string

func (e ErrInvalid) Error() string { return fmt.Sprintf("invalid login for user %#q", string(e)) }

func CheckLoginExists(l *Login) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		err := store.CheckExists(LoginBucket, []byte(l.Name))(tx)
		if store.IsMissing(err) {
			return ErrMissing(l.Name)
		}
		return err
	}
}

func CheckLoginNotExist(l *Login) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		err := store.CheckNotExist(LoginBucket, []byte(l.Name))(tx)
		if store.IsExists(err) {
			return ErrExists(l.Name)
		}
		return err
	}
}

func ValidateNew(l *Login) error {
	if lp := len(l.PWHash); lp != sha256.Size {
		return errors.Errorf("invalid SHA-256 pwhash: len is "+
			"%d (must be %d bytes, as base64 encoded string)",
			lp,
			sha256.Size,
		)
	}

	return users.ValidateNew(&(l.User))
}

func Check(l *Login) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		got := new(Login)
		err := store.Unmarshal(LoginBucket, got, []byte(l.Name))(tx)
		if err != nil {
			return err
		}
		cmp := sha256.Sum256(append(l.PWHash, got.Salt.Bytes()...))
		if !bytes.Equal(got.PWHash, cmp[:]) {
			return ErrInvalid(l.Name)
		}
		return nil
	}
}

func Create(l *Login, salt uuid.UUID) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		hash := sha256.Sum256(append(l.PWHash, salt.Bytes()...))
		toStore := &Login{
			User:   l.User,
			PWHash: hash[:],
			Salt:   salt,
		}

		bs, err := json.Marshal(toStore)
		if err != nil {
			return err
		}

		return store.Put(LoginBucket, []byte(l.Name), bs)(tx)
	}
}

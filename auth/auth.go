package auth

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"

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
	Disabled bool      `json:"disabled,omitempty"`
	PWHash   []byte    `json:"pwhash"`
	Salt     uuid.UUID `json:"salt"`
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

func Check(l *Login) store.View {
	return func(tx *bolt.Tx) error {
		got := new(Login)
		err := store.Unmarshal(LoginBucket, got, []byte(l.Name))(tx)
		switch {
		case store.IsMissing(err):
			return ErrMissing(l.Name)
		case err != nil:
			return err
		case got.Disabled:
			return ErrDisabled(l.Name)
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

type tokens struct {
	ts []Token
	rs []Token
}

func (c *tokens) Find(userID string) store.View {
	return store.ForEach(ContextBucket, func(_, v []byte) error {
		var into Context
		if err := json.Unmarshal(v, &into); err != nil {
			return err
		}

		if into.UserID == userID {
			c.ts = append(c.ts, into.Token)
			c.rs = append(c.rs, into.RefreshToken)
		}

		// TODO: Why am I broken?  I find the things, but other
		// callers don't see them in me any more.

		return nil
	})
}

func (c *tokens) DeleteRefresh(tx *bolt.Tx) error {
	b := tx.Bucket(RefreshBucket)
	for _, r := range c.rs {
		if err := b.Delete(r); err != nil {
			return err
		}
	}

	return nil
}

func (c *tokens) DeleteSessions(tx *bolt.Tx) error {
	b := tx.Bucket(SessionBucket)
	for _, t := range c.ts {
		if err := b.Delete(t); err != nil {
			return err
		}
	}

	return nil
}

func (c *tokens) DeleteContexts(tx *bolt.Tx) error {
	b := tx.Bucket(ContextBucket)
	for _, t := range c.ts {
		if err := b.Delete(t); err != nil {
			return err
		}
	}

	return nil
}

func Disable(userID string) store.Mutation {
	all := new(tokens)
	return store.Wrap(
		disableLogin(userID),
		all.Find(userID),
		all.DeleteRefresh,
		all.DeleteSessions,
		all.DeleteContexts,
	)
}

func disableLogin(userID string) store.Mutation {
	return func(tx *bolt.Tx) error {
		into := new(Login)
		bID := []byte(userID)
		err := store.Unmarshal(LoginBucket, into, bID)(tx)
		switch {
		case store.IsMissing(err):
			return nil
		case err != nil:
			return err
		}

		into.Disabled = true
		return store.Marshal(LoginBucket, into, bID)(tx)
	}
}

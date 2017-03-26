package auth

import (
	"encoding/json"
	"fmt"

	"github.com/synapse-garden/sg-proto/store"

	"github.com/boltdb/bolt"
)

const (
	CtxToken CtxField = iota
	CtxRefreshToken
	CtxUserID
)

type CtxField int

var (
	ContextBucket = store.Bucket("contexts")
)

type ErrContextMissing []byte

func (e ErrContextMissing) Error() string {
	return fmt.Sprintf("session context not found for %#q", string(e))
}

// IsContextMissing indicates whether the given error is an
// ErrContextMissing.
func IsContextMissing(e error) bool {
	if e == nil {
		return false
	}
	_, ok := e.(ErrContextMissing)
	return ok
}

// Context maps a Session token to other IDs which can be used to look
// up values from other buckets, or be threaded through headers by
// middleware, etc.
type Context struct {
	Token        Token
	RefreshToken Token
	UserID       string
}

func (c *Context) ByField(field CtxField) interface{} {
	return map[CtxField]interface{}{
		CtxToken:        c.Token,
		CtxRefreshToken: c.RefreshToken,
		CtxUserID:       c.UserID,
	}[field]
}

type Found Context

func (Found) Error() string { return "" }

// FindContext retrieves a context by UserID.  This might take a while
// if there are a lot of stored contexts.
func FindContext(id string) store.Mutation {
	return func(tx *bolt.Tx) error {
		ctx := new(Context)
		err := tx.Bucket(ContextBucket).ForEach(
			func(k, v []byte) error {
				if err := json.Unmarshal(v, ctx); err != nil {
					return err
				}
				if ctx.UserID == id {
					return Found(*ctx)
				}

				return nil
			})

		return err
	}
}

func SaveContext(c *Context) func(*bolt.Tx) error {
	return store.Marshal(ContextBucket, c, c.Token)
}

func GetContext(c *Context, t Token) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		err := store.Unmarshal(ContextBucket, c, t)(tx)
		if store.IsMissing(err) {
			return ErrContextMissing(t)
		}
		return err
	}
}

func DeleteContext(t Token) func(*bolt.Tx) error {
	return store.Delete(ContextBucket, t)
}

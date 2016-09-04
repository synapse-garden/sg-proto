package auth

import (
	"fmt"

	"github.com/boltdb/bolt"
	"github.com/synapse-garden/sg-proto/store"
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

package auth

import (
	"fmt"
	"time"

	"github.com/boltdb/bolt"
	uuid "github.com/satori/go.uuid"
	"github.com/synapse-garden/sg-proto/store"
)

// Expiration is how long a session takes to expire.
const Expiration = 5 * time.Minute

type TokenType int

const Bearer TokenType = iota

var (
	SessionBucket = store.Bucket("sessions")
	RefreshBucket = store.Bucket("refresh")
)

type Token []byte

type ErrMissingSession []byte

func (e ErrMissingSession) Error() string {
	return fmt.Sprintf("no such session %#q", string(e))
}

type ErrTokenExpired []byte

func (e ErrTokenExpired) Error() string {
	return fmt.Sprintf("session %#q expired", string(e))
}

// Session is a client login session.
type Session struct {
	Token        Token         `json:"token"`
	ExpiresIn    time.Duration `json:"expires_in"`
	Expiration   time.Time     `json:"expires_at,omitempty"`
	TokenType    TokenType     `json:"token_type"`
	RefreshToken Token         `json:"refresh_token"`
}

func NewToken(t TokenType) []byte {
	switch t {
	case Bearer:
		return uuid.NewV4().Bytes()
	}

	return nil
}

func RefreshIfValid(
	s *Session,
	expires time.Time,
	validFor time.Duration,
) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		err := store.CheckExists(
			RefreshBucket,
			s.RefreshToken,
		)(tx)
		switch {
		case store.IsMissing(err):
			return ErrMissingSession(s.RefreshToken)
		case err != nil:
			return err
		}
		return Refresh(s, expires, validFor)(tx)
	}
}

func Refresh(
	s *Session,
	expires time.Time,
	validFor time.Duration,
) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) (err error) {
		var (
			oldExpiresIn  = s.ExpiresIn
			oldExpiration = s.Expiration
		)
		s.Expiration = expires
		s.ExpiresIn = validFor

		defer func() {
			if err != nil {
				s.ExpiresIn = oldExpiresIn
				s.Expiration = oldExpiration
			}
		}()

		return store.Marshal(SessionBucket, s, s.Token)(tx)
	}
}

// CheckToken attempts to load the given Token's Session from the
// Sessions bucket.  If it was missing, it returns ErrMissingSession.
// If it was expired, it returns ErrTokenExpired.  This functionality
// should be used to branch logic into a RefreshIfValid.  The REST API
// user should not be trusted with the knowledge that a given token ever
// existed.  From the REST API user's point of view, an expired session
// with an invalid refresh token simply does not exist.
func CheckToken(s *Session) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		var (
			now      = time.Now().UTC()
			key      = s.Token
			existent = new(Session)
		)
		err := store.Unmarshal(SessionBucket, existent, key)(tx)

		switch {
		case store.IsMissing(err):
			// The session was not found.
			return ErrMissingSession(key)
		case err != nil:
			// There was an unknown error.
			return err
		case existent.Expiration.Before(now):
			// The token has already expired.
			return ErrTokenExpired(key)
		}

		return nil
	}
}

// NewSession prepares and assigns values to the given Session, and
// stores them in the database, or returns any error.
func NewSession(
	s *Session,
	expires time.Time,
	validFor time.Duration,
	token, refresh Token,
) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) (err error) {
		var (
			expiration = expires.Add(validFor)
			kind       = Bearer

			oldToken, oldRefresh = s.Token, s.RefreshToken
			oldExpiration        = s.Expiration
			oldExpiresAt         = s.ExpiresIn
			oldType              = s.TokenType
		)

		s.Token = token
		s.RefreshToken = refresh
		s.TokenType = kind
		s.Expiration = expiration
		s.ExpiresIn = validFor

		defer func() {
			if err != nil {
				s.Token = oldToken
				s.RefreshToken = oldRefresh
				s.Expiration = oldExpiration
				s.ExpiresIn = oldExpiresAt
				s.TokenType = oldType
			}
		}()

		err = store.Marshal(SessionBucket, s, s.Token)(tx)
		if err != nil {
			return
		}
		return store.Put(RefreshBucket, s.RefreshToken, nil)(tx)
	}
}

func DeleteSession(s *Session) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		err := store.Delete(SessionBucket, s.Token)(tx)
		if store.IsMissing(err) {
			return nil
		} else if err != nil {
			return err
		}

		err = store.Delete(RefreshBucket, s.RefreshToken)(tx)
		if store.IsMissing(err) {
			return nil
		} else if err != nil {
			return err
		}
		return nil
	}
}

package auth_test

import (
	"github.com/synapse-garden/sg-proto/auth"
	. "gopkg.in/check.v1"
)

func (s *AuthSuite) TestErrMissingSessionError(c *C) {
	c.Check(
		auth.ErrMissingSession("hello").Error(),
		Equals,
		"no such session `hello`",
	)
}

func (s *AuthSuite) TestErrTokenExpiredError(c *C) {
	c.Check(
		auth.ErrTokenExpired("hello").Error(),
		Equals,
		"session `hello` expired",
	)
}

func (s *AuthSuite) TestCheckToken(c *C) {
	// If the token is not in the database, return errmissing.
	// If there was an unknown error, return it.
	// If the token's expiration is before now, return errexpired.
	// Otherwise (it's current and present) return nil.
}

func (s *AuthSuite) TestNewSession(c *C) {
	// If an error occurred in store.Marshal, reset the values of
	// the given Session.

	// Otherwise, the new session with the new values should be
	// stored, and it should conform to the expected new values.
}

func (s *AuthSuite) TestNewToken(c *C) {
	// NewToken should generate a new v4 UUID dot Bytes for Bearer.
	// Otherwise, it should return nil.
}

func (s *AuthSuite) TestRefreshIfValid(c *C) {
	// If the given Refresh Token is not present in RefreshBucket,
	// return ErrMissingSession.  If an unknown error occurred,
	// return it.  Otherwise, call Refresh with the passed values.
}

func (s *AuthSuite) TestRefresh(c *C) {
	// Marshal the given Session into SessionBucket with the new
	// expiration and validFor, and the same Token.
}

func (s *AuthSuite) TestDeleteSession(c *C) {
	// DeleteSession should Delete the given Session from
	// SessionBucket, and the given Session's Refresh Token from
	// RefreshBucket.  If either value is not present, it should
	// not complain.
}

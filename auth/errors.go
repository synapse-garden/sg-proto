package auth

import "fmt"

func IsExists(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(ErrExists)
	return ok
}

type ErrExists string

func (e ErrExists) Error() string { return fmt.Sprintf("login for user %#q already exists", string(e)) }

type ErrMissing string

func (e ErrMissing) Error() string { return fmt.Sprintf("login for user %#q not found", string(e)) }

type ErrInvalid string

func (e ErrInvalid) Error() string { return fmt.Sprintf("invalid login for user %#q", string(e)) }

type ErrInvalidTokenType string

func (e ErrInvalidTokenType) Error() string {
	return fmt.Sprintf("invalid token type %q", string(e))
}

func IsInvalidTokenType(e error) bool {
	if e == nil {
		return false
	}
	_, ok := e.(ErrInvalidTokenType)
	return ok
}

type ErrMissingSession []byte

func (e ErrMissingSession) Error() string {
	return fmt.Sprintf("no such session %#q", string(e))
}

func IsMissingSession(e error) bool {
	if e == nil {
		return false
	}
	_, ok := e.(ErrMissingSession)
	return ok
}

type ErrTokenExpired []byte

func (e ErrTokenExpired) Error() string {
	return fmt.Sprintf("session %#q expired", string(e))
}

func IsTokenExpired(e error) bool {
	if e == nil {
		return false
	}
	_, ok := e.(ErrTokenExpired)
	return ok
}

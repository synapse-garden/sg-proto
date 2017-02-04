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

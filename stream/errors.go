package stream

import "fmt"

type errMissing string

func (e errMissing) Error() string {
	return fmt.Sprintf("no such stream %#q", string(e))
}

type errExists string

func (e errExists) Error() string {
	return fmt.Sprintf("stream %#q already exists", string(e))
}

type errUnauthorized string

func (e errUnauthorized) Error() string {
	return fmt.Sprintf("user %#q unauthorized", string(e))
}

func IsMissing(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(errMissing)
	return ok
}

func IsExists(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(errExists)
	return ok
}

func IsUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(errUnauthorized)
	return ok
}

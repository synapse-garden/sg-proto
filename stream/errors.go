package stream

import "fmt"

type errStreamMissing string

func (e errStreamMissing) Error() string {
	return fmt.Sprintf("no such stream %#q", string(e))
}

type errStreamExists string

func (e errStreamExists) Error() string {
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
	_, ok := err.(errStreamMissing)
	return ok
}

func IsExists(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(errStreamExists)
	return ok
}

func IsUnauthorized(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(errUnauthorized)
	return ok
}

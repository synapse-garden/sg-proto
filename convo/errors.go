package convo

import "fmt"

type errMissing string

func (e errMissing) Error() string {
	return fmt.Sprintf("no such convo %#q", string(e))
}

type errExists string

func (e errExists) Error() string {
	return fmt.Sprintf("convo %#q already exists", string(e))
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

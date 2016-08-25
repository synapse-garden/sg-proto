package store

import "fmt"

type ExistsError []byte

func (e ExistsError) Error() string {
	return fmt.Sprintf("user %#q already exists", e)
}

func IsExists(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(ExistsError)
	return ok
}

type MissingError []byte

func (m MissingError) Error() string {
	return fmt.Sprintf("no such key %#q", []byte(m))
}

func IsMissing(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(MissingError)
	return ok
}

package store

import "fmt"

type KeyError struct {
	Key, Bucket []byte
}

type ExistsError KeyError

func (e ExistsError) Error() string {
	return fmt.Sprintf("key %#q already exists in bucket %#q", e.Key, e.Bucket)
}

func IsExists(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*ExistsError)
	return ok
}

type MissingError KeyError

func (m MissingError) Error() string {
	return fmt.Sprintf("key %#q not in bucket %#q", m.Key, m.Bucket)
}

func IsMissing(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(*MissingError)
	return ok
}

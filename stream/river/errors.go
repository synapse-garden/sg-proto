package river

import "fmt"

type errExists string

// Error implements error on errRiverExists.
func (e errExists) Error() string {
	return fmt.Sprintf("river %#q already exists", string(e))
}

// IsExists returns true if the error is an existent River when one is
// not expected.
func IsExists(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(errExists)
	return ok
}

type errStreamMissing string

func (e errStreamMissing) Error() string {
	return fmt.Sprintf("no such stream %#q", string(e))
}

type errStreamExists string

func (e errStreamExists) Error() string {
	return fmt.Sprintf("stream %#q already exists", string(e))
}

type errUnknownSurvey []byte

func (e errUnknownSurvey) Error() string {
	return fmt.Sprintf("unkwown survey %#q", []byte(e))
}

func IsStreamMissing(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(errStreamMissing)
	return ok
}

func IsStreamExists(err error) bool {
	if err == nil {
		return false
	}
	_, ok := err.(errStreamExists)
	return ok
}

// IsUnknownSurvey returns true if the error is errUnknownSurvey.
func IsUnknownSurvey(err error) bool {
	if err == nil {
		return false
	}

	_, ok := err.(errUnknownSurvey)
	return ok
}

package testing

import "time"

// Timeout constants.
const (
	ShortWait    = 20 * time.Millisecond
	LongWait     = 300 * time.Millisecond
	CleanupWait  = 10 * time.Millisecond
	VeryLongWait = 3 * time.Second
)

// Timer implements util.Timer using a given Time.  Now() will always
// return that Time value.
type Timer time.Time

// Now implements Timer.Now on Timer.
func (t Timer) Now() time.Time {
	return time.Time(t)
}

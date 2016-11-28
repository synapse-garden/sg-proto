package util

import "time"

// Timer is an interface which can wrap various internal timers instead
// of using the global time.Now() function.
type Timer interface {
	Now() time.Time
}

// SimpleTimer implements Timer using time.Now.
type SimpleTimer struct{}

// Now implements Timer.Now on SimpleTimer.
func (SimpleTimer) Now() time.Time {
	return time.Now()
}

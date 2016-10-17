package testing

import "time"

// Timeout constants.
const (
	ShortWait    = 20 * time.Millisecond
	LongWait     = 300 * time.Millisecond
	CleanupWait  = 10 * time.Millisecond
	VeryLongWait = 3 * time.Second
)

package task

import (
	"time"

	"github.com/synapse-garden/sg-proto/users"
)

type Filter interface {
	Member(*Task) bool
}

type Completion int

const (
	Incomplete Completion = iota
	Complete
)

func (c Completion) Member(of *Task) bool {
	switch c {
	case Incomplete:
		return !of.Completed
	case Complete:
		return of.Completed
	default:
		return false
	}
}

type Overdue time.Time

func (o Overdue) Member(of *Task) bool {
	return of.Due != nil &&
		!of.Completed &&
		of.Due.Before(time.Time(o))
}

type NotYetDue time.Time

func (n NotYetDue) Member(of *Task) bool {
	d := of.Due
	if d == nil {
		return true
	}

	return of.Due.After(time.Time(n))
}

type DueWithin struct {
	From time.Time
	Til  time.Time
}

func (dw DueWithin) Member(of *Task) bool {
	d := of.Due
	if d == nil {
		return false
	}

	return d.After(dw.From) && d.Before(dw.Til)
}

type Not struct{ Filter }

func (n Not) Member(of *Task) bool { return n.Filter.Member(of) }

type ByOwner users.ByOwner

func (b ByOwner) Member(of *Task) bool {
	return users.ByOwner(b).Member(of.Group)
}

// ByReader is a Filter for Tasks that have the given read user.
type ByReader string

// Member implements Filter on ByReader.
func (b ByReader) Member(of *Task) bool {
	return users.ByReader(b).Member(of.Group)
}

// ByWriter is a Filter for Tasks that have the given read user.
type ByWriter string

// Member implements Filter on ByWriter.
func (b ByWriter) Member(of *Task) bool {
	return users.ByWriter(b).Member(of.Group)
}

// MultiAnd applies multiple Filters which all must be true.
type MultiAnd []Filter

// Member implements Filter on MultiAnd.
func (m MultiAnd) Member(s *Task) bool {
	for _, f := range []Filter(m) {
		if !f.Member(s) {
			return false
		}
	}
	// All passed.
	return true
}

// MultiOr applies multiple Filters, any of which may be true.
type MultiOr []Filter

// Member implements Filter on MultiOr.
func (m MultiOr) Member(s *Task) bool {
	for _, f := range []Filter(m) {
		if f.Member(s) {
			return true
		}
	}
	// None passed.
	return false
}

package client

import (
	"bufio"
	"io"

	"github.com/synapse-garden/sg-proto/auth"
)

// State models a Client's side effects, including buffered output.
type State struct {
	Session *auth.Session

	*bufio.Writer
	*bufio.Scanner
}

var (
	_ = io.ReaderFrom(State{})
	_ = io.Writer(State{})
)

// MakeState returns a new State made from the given Reader and Writer.
func MakeState(w io.Writer, r io.Reader) State {
	return State{
		Session: new(auth.Session),
		Writer:  bufio.NewWriter(w),
		Scanner: bufio.NewScanner(r),
	}
}

// ReadFrom implements ReaderFrom on State using its underlying Writer,
// which is flushed after reading.
func (s State) ReadFrom(r io.Reader) (int64, error) {
	n, err := s.Writer.ReadFrom(r)
	if err != nil {
		return n, err
	}
	return n, s.Writer.Flush()
}

// Write implements io.Writer on State using its underlying Writer,
// which is flushed after writing.
func (s State) Write(bs []byte) (int, error) {
	n, err := s.Writer.Write(bs)
	if err != nil {
		return n, err
	}
	return int(n), s.Writer.Flush()
}

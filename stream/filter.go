package stream

// Filter determines whether the given Stream is a member of a set.
type Filter interface {
	Member(*Stream) bool
}

// ByOwner is a Filter for Steams that have the given owner.
type ByOwner string

// Member implements Filter on ByOwner.
func (b ByOwner) Member(s *Stream) bool {
	return s.Owner == string(b)
}

// ByReader is a Filter for Steams that have the given read user.
type ByReader string

// Member implements Filter on ByReader.
func (b ByReader) Member(s *Stream) bool {
	return s.Readers[string(b)]
}

// ByWriter is a Filter for Steams that have the given read user.
type ByWriter string

// Member implements Filter on ByWriter.
func (b ByWriter) Member(s *Stream) bool {
	return s.Writers[string(b)]
}

// MultiAnd applies multiple Filters which all must be true.
type MultiAnd []Filter

// Member implements Filter on MultiAnd.
func (m MultiAnd) Member(s *Stream) bool {
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
func (m MultiOr) Member(s *Stream) bool {
	for _, f := range []Filter(m) {
		if f.Member(s) {
			return true
		}
	}
	// None passed.
	return false
}
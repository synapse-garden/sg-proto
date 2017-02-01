package users

// Group is a set of users which have different membership levels.
type Group struct {
	Owner string `json:"owner"`

	Readers map[string]bool `json:"readers"`
	Writers map[string]bool `json:"writers"`
}

// DiffGroups returns a bool map where removed users' names are keys to
// false values, and added or retained users' names are keys to true
// values.  It is assumed all Writers are also Readers and the Owner is
// unchanged.
func DiffGroups(old, new Group) map[string]bool {
	was := make(map[string]bool)
	allOld, allNew := AllUsers(old), AllUsers(new)

	for u := range allOld {
		was[u] = allNew[u]
	}
	for u := range allNew {
		was[u] = true
	}

	return was
}

// AllUsers gets a map of all users in the Group.
func AllUsers(g Group) map[string]bool {
	all := map[string]bool{g.Owner: true}
	for u := range g.Readers {
		all[u] = true
	}
	for u := range g.Writers {
		all[u] = true
	}
	return all
}

// Filter determines Group membership.
type Filter interface {
	Member(Group) bool
}

// ByOwner is a Filter for Groups that have the given owner.
type ByOwner string

// Member implements Filter on ByOwner.
func (b ByOwner) Member(s Group) bool {
	return s.Owner == string(b)
}

// ByReader is a Filter for Groups that have the given read user.
type ByReader string

// Member implements Filter on ByReader.
func (b ByReader) Member(s Group) bool {
	return s.Readers[string(b)]
}

// ByWriter is a Filter for Groups that have the given read user.
type ByWriter string

// Member implements Filter on ByWriter.
func (b ByWriter) Member(s Group) bool {
	return s.Writers[string(b)]
}

// MultiAnd applies multiple Filters which all must be true.
type MultiAnd []Filter

// Member implements Filter on MultiAnd.
func (m MultiAnd) Member(s Group) bool {
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
func (m MultiOr) Member(s Group) bool {
	for _, f := range []Filter(m) {
		if f.Member(s) {
			return true
		}
	}
	// None passed.
	return false
}

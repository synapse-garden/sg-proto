package users_test

import (
	"github.com/synapse-garden/sg-proto/users"

	. "gopkg.in/check.v1"
)

var (
	_ = users.Filter(users.ByOwner(""))
	_ = users.Filter(users.ByReader(""))
	_ = users.Filter(users.MultiOr(nil))
	_ = users.Filter(users.MultiAnd(nil))
)

func (s *UsersSuite) TestGroupDiff(c *C) {
	for i, test := range []struct {
		givenFrom, givenTo users.Group
		expect             map[string]bool
	}{{
		givenFrom: users.Group{Owner: "x"},
		givenTo:   users.Group{Owner: "x"},
		expect:    map[string]bool{"x": true},
	}, {
		givenFrom: users.Group{
			Owner:   "x",
			Readers: map[string]bool{"bodie": true},
		},
		givenTo: users.Group{
			Owner: "y",
		},
		expect: map[string]bool{
			"x": false, "y": true, "bodie": false,
		},
	}, {
		givenFrom: users.Group{Owner: "x"},
		givenTo: users.Group{
			Owner:   "x",
			Readers: map[string]bool{"bodie": true},
		},
		expect: map[string]bool{"x": true, "bodie": true},
	}, {
		givenFrom: users.Group{
			Owner:   "x",
			Readers: map[string]bool{"bodie": true},
		},
		givenTo: users.Group{
			Owner:   "x",
			Readers: map[string]bool{"bodie": true},
		},
		expect: map[string]bool{"x": true, "bodie": true},
	}, {
		givenFrom: users.Group{
			Owner:   "x",
			Readers: map[string]bool{"bodie": true},
		},
		givenTo: users.Group{
			Owner:   "x",
			Readers: map[string]bool{"bodie": true},
		},
		expect: map[string]bool{"x": true, "bodie": true},
	}, {
		givenFrom: users.Group{
			Owner: "x",
			Readers: map[string]bool{
				"bodie": true, "bob": true,
			},
		},
		givenTo: users.Group{
			Owner:   "x",
			Readers: map[string]bool{"bodie": true},
		},
		expect: map[string]bool{
			"x": true, "bodie": true, "bob": false,
		},
	}, {
		givenFrom: users.Group{
			Owner:   "x",
			Readers: map[string]bool{"bodie": true},
		},
		givenTo: users.Group{
			Owner: "x",
			Readers: map[string]bool{
				"bodie": true, "bob": true,
			},
		},
		expect: map[string]bool{
			"x": true, "bodie": true, "bob": true,
		},
	}, {
		givenFrom: users.Group{
			Owner: "x",
			Readers: map[string]bool{
				"bodie": true, "jim": true,
			},
		},
		givenTo: users.Group{
			Owner: "x",
			Readers: map[string]bool{
				"bodie": true, "bob": true,
			},
		},
		expect: map[string]bool{
			"x": true, "bodie": true,
			"bob": true, "jim": false,
		},
	}} {
		c.Logf("test %d: expect %+v", i, test.expect)
		got := users.DiffGroups(test.givenFrom, test.givenTo)
		c.Check(got, DeepEquals, test.expect)
	}
}

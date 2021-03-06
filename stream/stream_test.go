package stream_test

import (
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	. "gopkg.in/check.v1"
)

var _ = store.Resourcer(stream.Stream{})
var _ = store.Resourcer(&stream.Stream{})

func checkStreamMatch(
	c *C,
	db *bolt.DB,
	id string,
	expect *stream.Stream,
) {
	str := new(stream.Stream)
	err := db.View(store.Unmarshal(
		stream.StreamBucket, str, []byte(id),
	))
	c.Check(err, IsNil)
	c.Check(str, DeepEquals, expect)
}

func (s *StreamSuite) TestCheckNotExist(c *C) {
	c.Check(s.db.View(stream.CheckNotExist("x")), IsNil)

	c.Assert(s.db.Update(stream.Upsert(&stream.Stream{
		Group: users.Group{
			Owner:   "bob",
			Readers: map[string]bool{"bob": true, "bart": true},
			Writers: map[string]bool{"bob": true, "bart": true},
		},

		ID: "x",
	})), IsNil)

	err := s.db.View(stream.CheckNotExist("x"))
	c.Check(err, ErrorMatches, "stream `x` already exists")
	c.Check(stream.IsExists(err), Equals, true)
}

func (s *StreamSuite) TestGet(c *C) {
	given := &stream.Stream{
		Group: users.Group{
			Owner:   "bob",
			Readers: map[string]bool{"bob": true, "bart": true},
			Writers: map[string]bool{"bob": true, "bart": true},
		},
		ID: "x",
	}
	c.Assert(s.db.Update(stream.Upsert(given)), IsNil)

	str := new(stream.Stream)
	err := s.db.View(stream.Get(str, "y"))
	c.Check(err, ErrorMatches, "no such stream `y`")
	c.Check(stream.IsMissing(err), Equals, true)

	checkStreamMatch(c, s.db, "x", given)
}

func (s *StreamSuite) TestUpsert(c *C) {
	given := &stream.Stream{
		Group: users.Group{
			Owner:   "bob",
			Readers: map[string]bool{"bob": true, "bart": true},
			Writers: map[string]bool{"bob": true, "bart": true},
		},

		ID: "x",
	}
	next := &stream.Stream{
		Group: users.Group{
			Owner:   "zed",
			Readers: map[string]bool{"bob": true, "bart": true},
			Writers: map[string]bool{"bob": true, "bart": true},
		},

		ID: "x",
	}
	c.Assert(s.db.Update(stream.Upsert(given)), IsNil)
	c.Assert(s.db.Update(stream.Upsert(next)), IsNil)

	checkStreamMatch(c, s.db, "x", next)
}

func (s *StreamSuite) TestDelete(c *C) {
	given := &stream.Stream{
		Group: users.Group{
			Owner:   "bob",
			Readers: map[string]bool{"bob": true, "bart": true},
			Writers: map[string]bool{"bob": true, "bart": true},
		},

		ID: "x",
	}
	c.Assert(s.db.Update(stream.Upsert(given)), IsNil)
	checkStreamMatch(c, s.db, "x", given)

	c.Assert(s.db.Update(stream.Delete(given.ID)), IsNil)

	c.Check(s.db.View(stream.CheckNotExist("x")), IsNil)
}

func (s *StreamSuite) TestGetAll(c *C) {
	var given = []*stream.Stream{
		{ID: "x0", Group: users.Group{Owner: "bob"}},
		{ID: "x1", Group: users.Group{
			Owner:   "bob",
			Readers: map[string]bool{"john": true},
		}}, {ID: "x2", Group: users.Group{
			Owner:   "bob",
			Readers: map[string]bool{"jim": true},
		}}, {ID: "x3", Group: users.Group{
			Owner: "bob",
			Readers: map[string]bool{
				"jim":  true,
				"john": true,
			},
		}},
		{ID: "x4", Group: users.Group{Owner: "john"}},
		{ID: "x5", Group: users.Group{
			Owner:   "john",
			Readers: map[string]bool{"bob": true},
		}}, {ID: "x6", Group: users.Group{
			Owner:   "john",
			Readers: map[string]bool{"jim": true},
		}}, {ID: "x7", Group: users.Group{
			Owner: "john",
			Readers: map[string]bool{
				"bob": true,
				"jim": true,
			},
		}},
		{ID: "x8", Group: users.Group{Owner: "jim"}},
	}
	for _, str := range given {
		c.Assert(s.db.Update(stream.Upsert(str)), IsNil)
	}

	for i, test := range []struct {
		user      string
		filters   []users.Filter
		expectErr string
		expect    []*stream.Stream
	}{{
		user: "bob",
		expect: []*stream.Stream{
			given[0], given[1], given[2], given[3],
			given[5], given[7],
		},
	}, {
		user:    "bob",
		filters: []users.Filter{users.ByReader("john")},
		expect:  []*stream.Stream{given[1], given[3]},
	}, {
		user:    "bob",
		filters: []users.Filter{users.ByReader("jim")},
		expect:  []*stream.Stream{given[2], given[3], given[7]},
	}, {
		user: "bob",
		filters: []users.Filter{
			users.ByReader("jim"), users.ByReader("john"),
		},
		expect: []*stream.Stream{given[3]},
	}, {
		user:    "bob",
		filters: []users.Filter{users.ByReader("james")},
	}, {
		user: "john",
		expect: []*stream.Stream{
			given[1], given[3], given[4],
			given[5], given[6], given[7],
		},
	}, {
		user:    "john",
		filters: []users.Filter{users.ByReader("bob")},
		expect:  []*stream.Stream{given[5], given[7]},
	}, {
		user:    "john",
		filters: []users.Filter{users.ByReader("jim")},
		expect:  []*stream.Stream{given[3], given[6], given[7]},
	}, {
		user: "john",
		filters: []users.Filter{
			users.ByReader("bob"), users.ByReader("jim"),
		},
		expect: []*stream.Stream{given[7]},
	}, {
		user:    "john",
		filters: []users.Filter{users.ByReader("james")},
	}, {
		user: "jim",
		expect: []*stream.Stream{
			given[2], given[3], given[6],
			given[7], given[8],
		},
	}, {
		user:    "jim",
		filters: []users.Filter{users.ByReader("john")},
		expect:  []*stream.Stream{given[3]},
	}} {
		c.Logf("test %d", i)
		var strs []*stream.Stream
		err := s.db.View(func(tx *bolt.Tx) (e error) {
			strs, e = stream.GetAll(
				test.user, test.filters...,
			)(tx)
			return
		})
		if test.expectErr != "" {
			c.Check(err, ErrorMatches, test.expectErr)
			continue
		}
		c.Check(err, IsNil)
		// expect := make(map[string]*stream.Stream)
		// actual := make(map[string]*stream.Stream)
		// for _, str := range test.expect {
		// 	expect[str.ID] = str
		// }
		// for _, str := range strs {
		// 	actual[str.ID] = str
		// }

		c.Check(strs, DeepEquals, test.expect)
	}
}

func (s *StreamSuite) BenchmarkAllStreams(c *C) {
	// c.Pause()
	// Make 100, 1k, 10k, 100k entries with e.g. 10% overlap
	// Scan under various search conditions
}

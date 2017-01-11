package task_test

import (
	"time"

	"github.com/synapse-garden/sg-proto/task"
	"github.com/synapse-garden/sg-proto/users"

	uuid "github.com/satori/go.uuid"
	. "gopkg.in/check.v1"
)

func makeUserMap(users []string) map[string]bool {
	ret := make(map[string]bool)
	for _, user := range users {
		ret[user] = true
	}
	return ret
}

func (s *TaskSuite) TestTaskStoreErr(c *C) {
	err := s.Update(task.ID(uuid.NewV4()).Store("hi"))
	c.Check(err, ErrorMatches, "unexpected Store arg of type string")
}

func (s *TaskSuite) TestTaskLoadErr(c *C) {
	err := s.View(task.ID(uuid.NewV4()).Load("hoho"))
	c.Check(err, ErrorMatches, "unexpected Load arg of type string")
}

func (s *TaskSuite) TestTaskStoreLoad(c *C) {
	someWhen := time.Now().Add(2 * time.Hour)
	for i, t := range []struct {
		should string

		owner, name      string
		readers, writers []string
		notes            []string
		due              *time.Time

		expectStoreErr string
		expectLoadErr  string
	}{{
		should:  "store and load with no notes",
		owner:   "bob",
		name:    "hello",
		readers: []string{"bob"},
		writers: []string{"bob"},
	}, {
		should:  "store and load OK with one note",
		owner:   "bob",
		name:    "goodbye",
		readers: []string{"bob"},
		writers: []string{"bob"},
		notes:   []string{"hello world"},
	}, {
		should:  "store and load OK with more notes",
		owner:   "bob",
		name:    "goodbye",
		readers: []string{"bob"},
		writers: []string{"bob"},
		notes:   []string{"hello world", "goodbye world"},
		due:     &someWhen,
	}} {
		c.Logf("test %d: %s", i, t.should)

		id := task.ID(uuid.NewV4())
		given := &task.Task{
			ID: id,

			Group: users.Group{
				Owner:   t.owner,
				Readers: makeUserMap(t.readers),
				Writers: makeUserMap(t.writers),
			},

			Name:  t.name,
			Notes: t.notes,

			Due: t.due,
		}
		// Note that "given" is mutated on Store, removing its
		// Notes and replacing them with Resources.  We make a
		// shallow copy here, preserving the Notes, before
		// assigning IDs.  The Load user should never see
		// non-nil Task.Resources.
		expect := new(task.Task)
		*expect = *given

		err := s.Update(id.Store(given))
		if e := t.expectStoreErr; e != "" {
			c.Check(err, ErrorMatches, e)
			continue
		}

		c.Assert(err, IsNil)
		c.Logf("task %#q stored with ID %s", given.Name, uuid.UUID(given.ID))

		got := new(task.Task)
		err = s.View(id.Load(got))
		if e := t.expectLoadErr; e != "" {
			c.Check(err, ErrorMatches, e)
			continue
		}

		c.Assert(err, IsNil)
		c.Check(got, DeepEquals, expect)
	}
}

package task_test

import (
	"os"
	"testing"

	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/task"
	sgt "github.com/synapse-garden/sg-proto/testing"
	"github.com/synapse-garden/sg-proto/text"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	. "gopkg.in/check.v1"
)

type TaskSuite struct {
	*bolt.DB

	tmpDir string
}

var _ = Suite(&TaskSuite{})

func Test(t *testing.T) { TestingT(t) }

func (s *TaskSuite) SetUpTest(c *C) {
	db, tmpDir, err := sgt.TempDB("sg-task-test")
	c.Assert(err, IsNil)
	c.Assert(db.Update(store.Wrap(
		store.Migrate(store.VerCurrent),
		store.SetupBuckets(
			users.UserBucket,
			task.TaskBucket,
			text.TextBucket,
		),
	)), IsNil)
	s.DB, s.tmpDir = db, tmpDir
}

func (s *TaskSuite) TearDownTest(c *C) {
	c.Assert(sgt.Cleanup(s.DB), IsNil)
	c.Assert(os.Remove(s.tmpDir), IsNil)
}

package convo_test

import (
	"os"
	"testing"

	"github.com/synapse-garden/sg-proto/convo"
	"github.com/synapse-garden/sg-proto/store"
	sgt "github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type ConvoSuite struct {
	db     *bolt.DB
	tmpDir string
}

var _ = Suite(&ConvoSuite{})

func (s *ConvoSuite) SetUpTest(c *C) {
	db, tmpDir, err := sgt.TempDB("sg-convo-test")
	c.Assert(err, IsNil)
	c.Assert(db.Update(store.Wrap(
		store.Migrate(store.VerCurrent),
		store.SetupBuckets(
			convo.ConvoBucket,
			convo.MessageBucket,
		),
	)), IsNil)
	s.db, s.tmpDir = db, tmpDir
}

func (s *ConvoSuite) TearDownTest(c *C) {
	if db := s.db; db != nil {
		c.Assert(sgt.Cleanup(db), IsNil)
		c.Assert(os.Remove(s.tmpDir), IsNil)
	}
}

package stream_test

import (
	"os"
	"testing"

	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream"
	sgt "github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type StreamSuite struct {
	db     *bolt.DB
	tmpDir string
}

var _ = Suite(&StreamSuite{})

func (s *StreamSuite) SetUpTest(c *C) {
	db, tmpDir, err := sgt.TempDB("sg-stream-test")
	c.Assert(err, IsNil)
	c.Assert(db.Update(store.Wrap(
		store.Migrate(store.VerCurrent),
		store.SetupBuckets(stream.StreamBucket),
	)), IsNil)
	s.db, s.tmpDir = db, tmpDir
}

func (s *StreamSuite) TearDownTest(c *C) {
	if db := s.db; db != nil {
		c.Assert(sgt.CleanupDB(db), IsNil)
		c.Assert(os.Remove(s.tmpDir), IsNil)
	}
}

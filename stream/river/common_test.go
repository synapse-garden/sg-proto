package river_test

import (
	"os"
	"testing"

	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream/river"
	sgt "github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type RiverSuite struct {
	db     *bolt.DB
	tmpDir string
}

var _ = Suite(&RiverSuite{})

func (s *RiverSuite) SetUpTest(c *C) {
	db, tmpDir, err := sgt.TempDB("sg-river-test")
	c.Assert(err, IsNil)
	c.Assert(db.Update(store.Wrap(
		store.Migrate(store.VerCurrent),
		store.SetupBuckets(river.RiverBucket),
	)), IsNil)
	s.db, s.tmpDir = db, tmpDir
}

func (s *RiverSuite) TearDownTest(c *C) {
	if db := s.db; db != nil {
		c.Assert(sgt.CleanupDB(db), IsNil)
		c.Assert(os.Remove(s.tmpDir), IsNil)
	}
}

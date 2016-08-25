package store_test

import (
	"errors"
	"os"
	"testing"

	"github.com/synapse-garden/sg-proto/store"
	sgt "github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type StoreSuite struct {
	db     *bolt.DB
	tmpDir string
}

var _ = Suite(&StoreSuite{})

func (s *StoreSuite) SetUpTest(c *C) {
	db, tmpDir, err := sgt.TempDB("sg-test")
	c.Assert(err, IsNil)
	c.Assert(db.Update(store.Wrap(
		store.Migrate(store.Version),
		store.SetupBuckets([]byte("foo")),
	)), IsNil)
	s.db, s.tmpDir = db, tmpDir
}

func (s *StoreSuite) TearDownTest(c *C) {
	if db := s.db; db != nil {
		c.Assert(sgt.CleanupDB(db), IsNil)
		c.Assert(os.Remove(s.tmpDir), IsNil)
	}
}
func (s *StoreSuite) TestMissingError(c *C) {
	m := store.MissingError("hello")
	c.Check(store.IsMissing(m), Equals, true)
	c.Check(store.IsMissing(errors.New("hello")), Equals, false)
	c.Check(m, ErrorMatches, "no such key `hello`")
}

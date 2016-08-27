package store_test

import (
	"testing"

	"github.com/boltdb/bolt"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type StoreSuite struct {
	db     *bolt.DB
	tmpDir string
}

var _ = Suite(&StoreSuite{})

func (s *StoreSuite) TestPrep(c *C) {
	// Wrap
	// Migrate
	// SetupBuckets
	// Put "version"
}

func (s *StoreSuite) TestSetupBuckets(c *C) {
	// TODO: Fill me out
}

func (s *StoreSuite) TestMigrate(c *C) {
	// if version bucket is nil, create version bucket
	// and put the version;
	// then compare to given version and MigrateFrom
}

func (s *StoreSuite) TestMigrateFrom(c *C) {
	// if same version, return nil
	// if version not found, return no migration defined
	// otherwise return the given migration
}

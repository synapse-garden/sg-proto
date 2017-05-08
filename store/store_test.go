package store_test

import (
	"os"
	tt "testing"

	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
)

func Test(t *tt.T) { TestingT(t) }

type StoreSuite struct {
	*bolt.DB

	tmpDir string
}

var _ = Suite(&StoreSuite{})

func (s *StoreSuite) SetUpTest(c *C) {
	var err error
	s.DB, s.tmpDir, err = testing.TempDB("store")
	c.Assert(err, IsNil)
}

func (s *StoreSuite) TearDownTest(c *C) {
	c.Assert(testing.Cleanup(s.DB), IsNil)
	c.Assert(os.Remove(s.tmpDir), IsNil)
}

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
	c.Log("Before store.Migrate, no version bucket.")
	c.Check(s.View(func(tx *bolt.Tx) error {
		if tx.Bucket(store.VersionBucket) != nil {
			return errors.New("found unexpected bucket")
		}
		return nil
	}), IsNil)
	c.Log("store.Migrate on a fresh DB fails for bad version.")
	c.Check(s.Update(store.Migrate(store.Version("boopty doopty"))),
		ErrorMatches,
		"no migration defined from version `` to `boopty doopty`",
	)
	c.Check(s.View(func(tx *bolt.Tx) error {
		if tx.Bucket(store.VersionBucket) != nil {
			return errors.New("found unexpected version bucket")
		}
		return nil
	}), IsNil)

	c.Log("store.Migrate on a fresh DB runs the VerCurrent migration.")
	c.Check(s.Update(store.Migrate(store.VerCurrent)), IsNil)
	c.Check(s.View(func(tx *bolt.Tx) error {
		b := tx.Bucket(store.VersionBucket)
		if b == nil {
			return errors.New("version bucket not found")
		}

		v := store.Version(b.Get([]byte("version")))
		if v != store.VerCurrent {
			return errors.Errorf("wrong version %q", v)
		}

		return nil
	}), IsNil)
}

func (s *StoreSuite) TestMigrateFrom(c *C) {
	// if same version, return nil
	// if version not found, return no migration defined
	// otherwise return the given migration
}

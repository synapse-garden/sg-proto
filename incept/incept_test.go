package incept_test

import (
	"errors"
	"os"
	"testing"

	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/store"
	sgt "github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
	uuid "github.com/satori/go.uuid"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type InceptSuite struct {
	tmpDir string
	db     *bolt.DB
}

var _ = Suite(&InceptSuite{})

func (s *InceptSuite) SetUpTest(c *C) {
	db, tmpDir, err := sgt.TempDB("sg-test")
	c.Assert(err, IsNil)
	c.Assert(db.Update(store.Wrap(
		store.Migrate(store.Version),
		store.SetupBuckets(incept.TicketBucket),
	)), IsNil)
	s.db, s.tmpDir = db, tmpDir
}

func (s *InceptSuite) TearDownTest(c *C) {
	if db := s.db; db != nil {
		c.Assert(sgt.CleanupDB(db), IsNil)
		c.Assert(os.Remove(s.tmpDir), IsNil)
	}
}

func (s *InceptSuite) TestMissingError(c *C) {
	m := incept.MissingError("hello")
	c.Check(incept.IsMissing(m), Equals, true)
	c.Check(incept.IsMissing(errors.New("hello")), Equals, false)
	c.Check(m, ErrorMatches, "no such ticket `hello`")
}

func (s *InceptSuite) TestingNewTicket(c *C) {
	var ks, vs [][]byte
	c.Assert(s.db.Update(func(tx *bolt.Tx) error {
		keys, vals, err := sgt.FindAll(tx, incept.TicketBucket)
		if err != nil {
			return err
		}
		for _, key := range keys {
			ks = append(ks, key)
		}
		for _, val := range vals {
			vs = append(vs, val)
		}
		return nil
	}), IsNil)
	c.Check(ks, IsNil)
	c.Check(vs, IsNil)
	c.Assert(s.db.Update(incept.NewTicket), IsNil)
	c.Assert(s.db.Update(func(tx *bolt.Tx) error {
		keys, vals, err := sgt.FindAll(tx, incept.TicketBucket)
		if err != nil {
			return err
		}
		for _, key := range keys {
			ks = append(ks, key)
		}
		for _, val := range vals {
			vs = append(vs, val)
		}
		return nil
	}), IsNil)

	c.Assert(len(ks), Equals, 1)
	_, err := uuid.FromBytes(ks[0])
	c.Check(err, IsNil)

	c.Check(vs, DeepEquals, [][]byte{{}})
}

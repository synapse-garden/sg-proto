package incept_test

import (
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

func (s *InceptSuite) TestNewTicket(c *C) {
	tkt := incept.Ticket(uuid.NewV4())
	c.Assert(s.db.View(assertNoTickets(c)), IsNil)
	c.Assert(s.db.Update(incept.NewTicket(tkt)), IsNil)
	c.Assert(s.db.View(assertTicketsExist(c, tkt)), IsNil)
}

func (s *InceptSuite) TestDeleteTicket(c *C) {
	tkt := incept.Ticket(uuid.NewV4())
	c.Assert(s.db.View(assertNoTickets(c)), IsNil)
	c.Assert(s.db.Update(incept.NewTicket(tkt)), IsNil)
	c.Assert(s.db.View(assertTicketsExist(c, tkt)), IsNil)
	c.Assert(s.db.Update(incept.DeleteTickets(tkt)), IsNil)
	c.Check(s.db.View(assertNoTickets(c)), IsNil)
}

func (s *InceptSuite) TestCheckKey(c *C) {}

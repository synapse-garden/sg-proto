package incept_test

import (
	"encoding/json"
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
		store.Migrate(store.VerCurrent),
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

func (s *InceptSuite) TestTicketMarshalJSON(c *C) {
	expect := uuid.NewV4().String()
	got, err := uuid.FromString(expect)
	c.Assert(err, IsNil)
	bs, err := json.Marshal(incept.Ticket(got))
	c.Assert(err, IsNil)
	c.Check(string(bs), Equals, `"`+expect+`"`)
}

func (s *InceptSuite) TestTicketUnmarshalJSON(c *C) {
	expect := incept.Ticket(uuid.NewV4())
	bs, err := json.Marshal(expect)
	c.Assert(err, IsNil)
	var got incept.Ticket
	c.Assert(json.Unmarshal(bs, &got), IsNil)
	c.Logf("%#q got\n%#q expect", got.String(), expect.String())
	c.Check(uuid.Equal(uuid.UUID(got), uuid.UUID(expect)), Equals, true)
}

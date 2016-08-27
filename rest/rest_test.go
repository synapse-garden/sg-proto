package rest_test

import (
	"os"
	"testing"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/store"
	sgt "github.com/synapse-garden/sg-proto/testing"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type RESTSuite struct {
	db      *bolt.DB
	tmpDir  string
	tickets []incept.Ticket
}

var (
	_ = Suite(&RESTSuite{})
)

func (s *RESTSuite) SetUpTest(c *C) {
	db, tmpDir, err := sgt.TempDB("sg-test")
	c.Assert(err, IsNil)
	c.Assert(db.Update(store.Wrap(
		store.Migrate(store.Version),
		store.SetupBuckets(
			incept.TicketBucket,
			users.UserBucket,
			auth.LoginBucket,
		),
	)), IsNil)
	s.db, s.tmpDir = db, tmpDir

	tkts := make([]incept.Ticket, 3)
	c.Assert(s.db.Update(func(tx *bolt.Tx) error {
		for i := range tkts {
			tkt, err := incept.NewTicket(tx)
			if err != nil {
				return err
			}
			tkts[i] = tkt
		}
		return nil
	}), IsNil)

	s.tickets = tkts
}

func (s *RESTSuite) TearDownTest(c *C) {
	if db := s.db; db != nil {
		c.Assert(sgt.CleanupDB(db), IsNil)
		c.Assert(os.Remove(s.tmpDir), IsNil)
	}
}

package rest_test

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/synapse-garden/sg-proto/admin"
	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream"
	sgt "github.com/synapse-garden/sg-proto/testing"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	uuid "github.com/satori/go.uuid"
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
			admin.AdminBucket,
			incept.TicketBucket,
			users.UserBucket,
			auth.LoginBucket,
			auth.SessionBucket,
			auth.RefreshBucket,
			auth.ContextBucket,
			stream.StreamBucket,
			stream.RiverBucket,
		),
	)), IsNil)
	s.db, s.tmpDir = db, tmpDir

	tkts := make([]incept.Ticket, 3)
	for i := range tkts {
		tkts[i] = incept.Ticket(uuid.NewV4())
	}
	c.Assert(s.db.Update(incept.NewTickets(tkts...)), IsNil)

	s.tickets = tkts
}

func (s *RESTSuite) TearDownTest(c *C) {
	runtime.Gosched()
	time.Sleep(sgt.CleanupWait)
	if db := s.db; db != nil {
		c.Assert(sgt.CleanupDB(db), IsNil)
		c.Assert(os.Remove(s.tmpDir), IsNil)
	}
}

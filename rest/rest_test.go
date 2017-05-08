package rest_test

import (
	"os"
	"runtime"
	"testing"
	"time"

	"github.com/synapse-garden/sg-proto/admin"
	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/convo"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream"
	"github.com/synapse-garden/sg-proto/stream/river"
	"github.com/synapse-garden/sg-proto/task"
	sgt "github.com/synapse-garden/sg-proto/testing"
	"github.com/synapse-garden/sg-proto/text"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	uuid "github.com/satori/go.uuid"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type RESTSuite struct {
	db     *bolt.DB
	tmpDir string
}

var _ = Suite(new(RESTSuite))

func (s *RESTSuite) SetUpTest(c *C) {
	db, tmpDir, err := sgt.TempDB("sg-test")
	c.Assert(err, IsNil)
	c.Assert(db.Update(store.Wrap(
		store.Migrate(store.VerCurrent),
		store.SetupBuckets(
			admin.AdminBucket,
			incept.TicketBucket,
			users.UserBucket,
			auth.LoginBucket,
			auth.SessionBucket,
			auth.RefreshBucket,
			auth.ContextBucket,
			stream.StreamBucket,
			river.RiverBucket,
			convo.ConvoBucket,
			convo.MessageBucket,
			convo.ScribeBucket,
			text.TextBucket,
			task.TaskBucket,
		),
		auth.ClearSessions,
		river.ClearRivers,
	)), IsNil)
	s.db, s.tmpDir = db, tmpDir
}

func prepareTickets(c *C, db *bolt.DB) []incept.Ticket {
	tkts := make([]incept.Ticket, 3)
	for i := range tkts {
		tkts[i] = incept.Ticket(uuid.NewV4())
	}
	c.Assert(db.Update(incept.NewTickets(tkts...)), IsNil)

	return tkts
}

func (s *RESTSuite) TearDownTest(c *C) {
	runtime.Gosched()
	time.Sleep(sgt.CleanupWait)
	if db := s.db; db != nil {
		c.Assert(sgt.Cleanup(db), IsNil)
		c.Assert(os.Remove(s.tmpDir), IsNil)
	}
}

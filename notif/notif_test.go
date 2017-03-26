package notif_test

import (
	"encoding/base64"
	"os"
	"testing"

	"github.com/synapse-garden/sg-proto/notif"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream/river"
	sgt "github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
	blake2b "github.com/minio/blake2b-simd"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type NotifSuite struct {
	db     *bolt.DB
	tmpDir string
}

var (
	_ = Suite(&NotifSuite{})
	_ = river.Topic(notif.UserTopic{})
)

func (s *NotifSuite) SetUpTest(c *C) {
	db, tmpDir, err := sgt.TempDB("sg-test")
	c.Assert(err, IsNil)
	c.Assert(db.Update(store.Wrap(
		store.Migrate(store.VerCurrent),
		store.SetupBuckets(river.RiverBucket),
	)), IsNil)
	s.db, s.tmpDir = db, tmpDir
}

func (s *NotifSuite) TearDownTest(c *C) {
	if db := s.db; db != nil {
		c.Assert(sgt.CleanupDB(db), IsNil)
		c.Assert(os.Remove(s.tmpDir), IsNil)
	}
}

func (s *NotifSuite) TestMakeUserTopic(c *C) {
	// Given id, topic is BLAKE2 64-bit hash of id.
	given1, given2 := "given1User", "given2User"
	expect1 := blake2b.Sum512([]byte(given1))
	expect2 := blake2b.Sum512([]byte(given2))

	c.Check(notif.MakeUserTopic(given1), DeepEquals, notif.UserTopic(expect1))
	c.Check(notif.MakeUserTopic(given2), DeepEquals, notif.UserTopic(expect2))
}

func (s *NotifSuite) TestUserTopicPrefix(c *C) {
	// Prefix should be package constant.
	given := notif.MakeUserTopic("givenUser")
	c.Check(given.Prefix(), Equals, notif.UserPrefix)
}

func (s *NotifSuite) TestUserTopicCode(c *C) {
	// Code should be USER+BLAKE2 64b hash of id.
	given := notif.MakeUserTopic("givenUser")
	sumExpect := blake2b.Sum512([]byte("givenUser"))
	expect := append([]byte{notif.UserPrefix}, sumExpect[:]...)

	c.Check(given.Code(), DeepEquals, expect)
}

func (s *NotifSuite) TestUserTopicName(c *C) {
	// Name should be "user "+base64 encoding of BLAKE2 64b hash of id.
	given := notif.MakeUserTopic("givenUser")
	sumExpect := blake2b.Sum512([]byte("givenUser"))
	expect := "user " + base64.StdEncoding.EncodeToString(sumExpect[:])

	c.Check(given.Name(), Equals, expect)
}

func (s *NotifSuite) TestUserTopicLen(c *C) {
	// Name should be "user "+base64 encoding of BLAKE2 64b hash of id.
	given := notif.MakeUserTopic("givenUser")
	sumExpect := blake2b.Sum512([]byte("givenUser"))

	c.Check(given.Len(), Equals, len(sumExpect)+1)
}

func (s *NotifSuite) TestTopics(c *C) {
	// Topics for a given user = a single UserTopic from his name.
	given := notif.Topics("givenUser")
	expect := []river.Topic{notif.MakeUserTopic("givenUser")}
	c.Check(given, DeepEquals, expect)
}

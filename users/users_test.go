package users_test

import (
	"os"
	tt "testing"

	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/testing"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	. "gopkg.in/check.v1"
)

type UsersSuite struct {
	*bolt.DB

	tmpDir string
}

var _ = Suite(&UsersSuite{})

func Test(t *tt.T) { TestingT(t) }

func (s *UsersSuite) SetUpTest(c *C) {
	var err error
	s.DB, s.tmpDir, err = testing.TempDB("users")
	c.Assert(err, IsNil)
	c.Assert(s.Update(store.SetupBuckets(
		users.UserBucket,
	)), IsNil)
}

func (s *UsersSuite) TearDownTest(c *C) {
	c.Assert(testing.CleanupDB(s.DB), IsNil)
	c.Assert(os.Remove(s.tmpDir), IsNil)
}

func (s *UsersSuite) TestAddCoin(c *C) {
	u := &users.User{}

	c.Log("AddCoin fails (store.IsMissing) for user with no name")
	err := s.Update(users.AddCoin(u, 5))
	c.Assert(err, ErrorMatches, "key `` not in bucket `users`")
	c.Check(store.IsMissing(err), Equals, true)

	c.Log("After failed AddCoin, DB is unmodified")
	c.Check(s.View(func(tx *bolt.Tx) error {
		ks, _, err := testing.FindAll(tx, users.UserBucket)
		if err != nil {
			return err
		}

		if len(ks) != 0 {
			return errors.Errorf("expected 0 keys, found %d", len(ks))
		}

		return nil
	}), IsNil)

	u.Name = "bob"
	u.Coin = 5

	c.Assert(s.Update(users.Create(u)), IsNil)
	c.Assert(s.Update(users.AddCoin(u, 3)), IsNil)
	c.Check(u.Coin, Equals, int64(8))
	c.Assert(s.View(func(tx *bolt.Tx) error {
		into := new(users.User)
		err := store.Unmarshal(users.UserBucket, into, []byte(u.Name))(tx)
		if err != nil {
			return err
		}

		c.Check(u, DeepEquals, into)
		return nil
	}), IsNil)
}

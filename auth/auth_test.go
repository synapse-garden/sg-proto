package auth_test

import (
	"crypto/sha256"
	"testing"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"

	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type AuthSuite struct {
	tmpDir string
	db     *bolt.DB
}

var _ = Suite(&AuthSuite{})

// func (s *AuthSuite) SetUpTest(c *C) {
// 	db, tmp, err := sgt.TempDB("auth")
// 	c.Assert(err, IsNil)
// 	c.Assert(db.Update(store.Prep(auth.LoginBucket)), IsNil)
//
// 	s.tmpDir, s.db = tmp, db
// }

// func (s *AuthSuite) TearDownTest(c *C) {
// 	if db := s.db; db != nil {
// 		c.Assert(sgt.CleanupDB(db), IsNil)
// 	}
// 	c.Assert(os.Remove(s.tmpDir), IsNil)
// }

func (s *AuthSuite) TestValidateNew(c *C) {
	realSha256Sum := sha256.Sum256([]byte("hello"))
	c.Logf(`the sha256 of "hello" is %#q\n`, realSha256Sum)
	for i, test := range []struct {
		Name   string
		PWHash []byte
		Coin   int64
		Expect string
	}{{
		Expect: `invalid SHA-256 pwhash: len is 0 \(must be 32 ` +
			`bytes, as base64 encoded string\)`,
	}, {
		PWHash: realSha256Sum[:],
		Expect: "name must not be blank",
	}, {
		PWHash: realSha256Sum[:],
		Name:   "bob",
		Coin:   3,
		Expect: "user cannot be created with coin",
	}, {
		PWHash: realSha256Sum[:],
		Name:   "bob",
	}} {
		c.Logf("test %d:\n  Name: %q\n  PWHash: %#q\n  Coin: %d",
			i,
			test.Name,
			test.PWHash,
			test.Coin,
		)

		err := auth.ValidateNew(&auth.Login{
			User: users.User{
				Name: test.Name,
				Coin: test.Coin,
			},
			PWHash: test.PWHash,
		},
		)
		if test.Expect == "" {
			c.Check(err, IsNil)
		} else {
			c.Check(err, ErrorMatches, test.Expect)
		}
	}
}

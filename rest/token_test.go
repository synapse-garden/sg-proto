package rest_test

import (
	// "github.com/synapse-garden/sg-proto/incept"
	// "github.com/synapse-garden/sg-proto/users"

	// "github.com/boltdb/bolt"
	. "gopkg.in/check.v1"
)

func (s *RESTSuite) TestToken(c *C) {
	// If a user's account does not exist, he cannot create a token.
	// If a user's account is disabled, he cannot create a token.
	// If a user's account is normal and enabled:
	//   - he can POST to /tokens to create a token
	//   - in order to do this, the POST body must be a valid
	//     auth.Login.
	//   - A valid auth.Login is one which is:
	//     > stored in auth.LoginBucket by Name
	//     > sha256( sha256( pw ) + stored.Salt ) == stored.PWHash
	//     > Not Disabled
	// c.Assert(s.db.Update(func(tx *bolt.Tx) error {
	// 	return incept.Incept()(tx)
	// }), IsNil)
	// for i, test := range []struct {
	// 	givenUser *users.User
	// 	expectErr string
	// }{{}} {
	// 	c.Logf("test %d")
	// 	r := htt.NewRecorder()
	// }
}

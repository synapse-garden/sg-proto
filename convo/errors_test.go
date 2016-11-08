package convo_test

import (
	"github.com/synapse-garden/sg-proto/convo"

	. "gopkg.in/check.v1"
)

func (s *ConvoSuite) TestErrMissing(c *C) {
	var err error
	c.Check(convo.IsMissing(err), Equals, false)
	err = convo.MakeMissingErr([]byte("b"))
	c.Check(err, ErrorMatches, "no such convo `b`")
	c.Check(convo.IsMissing(err), Equals, true)
}

func (s *ConvoSuite) TestErrExists(c *C) {
	var err error
	c.Check(convo.IsExists(err), Equals, false)
	err = convo.MakeExistsErr([]byte("b"))
	c.Check(err, ErrorMatches, "convo `b` already exists")
	c.Check(convo.IsExists(err), Equals, true)
}

func (s *ConvoSuite) TestErrUnauthorized(c *C) {
	var err error
	c.Check(convo.IsUnauthorized(err), Equals, false)
	err = convo.MakeUnauthorizedErr("bob")
	c.Check(err, ErrorMatches, "user `bob` unauthorized")
	c.Check(convo.IsUnauthorized(err), Equals, true)
}

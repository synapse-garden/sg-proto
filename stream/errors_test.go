package stream_test

import (
	"github.com/synapse-garden/sg-proto/stream"

	. "gopkg.in/check.v1"
)

func (s *StreamSuite) TestErrMissing(c *C) {
	var err error
	c.Check(stream.IsMissing(err), Equals, false)
	err = stream.MakeMissingErr([]byte("b"))
	c.Check(err, ErrorMatches, "no such stream `b`")
	c.Check(stream.IsMissing(err), Equals, true)
}

func (s *StreamSuite) TestErrExists(c *C) {
	var err error
	c.Check(stream.IsExists(err), Equals, false)
	err = stream.MakeExistsErr([]byte("b"))
	c.Check(err, ErrorMatches, "stream `b` already exists")
	c.Check(stream.IsExists(err), Equals, true)
}

func (s *StreamSuite) TestErrUnauthorized(c *C) {
	var err error
	c.Check(stream.IsUnauthorized(err), Equals, false)
	err = stream.MakeUnauthorizedErr("bob")
	c.Check(err, ErrorMatches, "user `bob` unauthorized")
	c.Check(stream.IsUnauthorized(err), Equals, true)
}

package river_test

import (
	"github.com/synapse-garden/sg-proto/stream/river"

	. "gopkg.in/check.v1"
)

func (s *RiverSuite) TestErrRiverExists(c *C) {
	var err error
	c.Check(river.IsExists(err), Equals, false)
	err = river.MakeRiverExistsErr("bob")
	c.Check(err, ErrorMatches, "river `bob` already exists")
	c.Check(river.IsExists(err), Equals, true)
}

func (s *RiverSuite) TestErrStreamMissing(c *C) {
	var err error
	c.Check(river.IsStreamMissing(err), Equals, false)
	err = river.MakeStreamMissingErr([]byte("b"))
	c.Check(err, ErrorMatches, "no such stream `b`")
	c.Check(river.IsStreamMissing(err), Equals, true)
}

func (s *RiverSuite) TestErrStreamExists(c *C) {
	var err error
	c.Check(river.IsStreamExists(err), Equals, false)
	err = river.MakeStreamExistsErr([]byte("b"))
	c.Check(err, ErrorMatches, "stream `b` already exists")
	c.Check(river.IsStreamExists(err), Equals, true)
}

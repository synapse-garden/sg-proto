package river_test

import (
	"github.com/synapse-garden/sg-proto/stream/river"
	. "gopkg.in/check.v1"
)

type someMissing struct{}

func (someMissing) IDs() []uint64 {
	return nil
}

func (someMissing) Error() string {
	return ""
}

func (s *RiverSuite) TestIsMissing(c *C) {
	e := someMissing{}
	var err error
	c.Check(river.IsMissing(e), Equals, true)
	c.Check(river.IsMissing(err), Equals, false)
}

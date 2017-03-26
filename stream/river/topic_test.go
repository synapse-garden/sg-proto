package river_test

import (
	"reflect"

	"github.com/synapse-garden/sg-proto/stream/river"

	. "gopkg.in/check.v1"
)

var _ = river.Topic(river.GlobalTopic{})
var _ = river.Topic(river.Global)

func (s *RiverSuite) TestGlobalTopic(c *C) {
	a := river.GlobalTopic{}
	c.Check(a.Code(), DeepEquals, []byte{0})
	c.Check(a.Name(), Equals, "global")
}

func (s *RiverSuite) TestGlobal(c *C) {
	c.Check(
		reflect.TypeOf(river.Global),
		Equals,
		reflect.TypeOf(river.GlobalTopic{}),
	)
}

func (s *RiverSuite) TestBytesFor(c *C) {
	got := river.BytesFor(testingTopic{pre: 0x3, code: []byte("hello")}, []byte("goodbyte"))
	expect := append([]byte{3}, []byte("hellogoodbyte")...)
	c.Check(got, DeepEquals, expect)
}

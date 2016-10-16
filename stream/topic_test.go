package stream_test

import (
	"reflect"

	"github.com/synapse-garden/sg-proto/stream"
	. "gopkg.in/check.v1"
)

var _ = stream.Topic(stream.GlobalTopic{})
var _ = stream.Topic(stream.Global)

func (s *StreamSuite) TestGlobalTopic(c *C) {
	a := stream.GlobalTopic{}
	c.Check(a.Code(), DeepEquals, []byte{0})
	c.Check(a.Name(), Equals, "global")
}

func (s *StreamSuite) TestGlobal(c *C) {
	c.Check(
		reflect.TypeOf(stream.Global),
		Equals,
		reflect.TypeOf(stream.GlobalTopic{}),
	)
}

func (s *StreamSuite) TestBytesFor(c *C) {
	got := stream.BytesFor(testingTopic{pre: 0x3, code: []byte("hello")}, []byte("goodbyte"))
	expect := append([]byte{3}, []byte("hellogoodbyte")...)
	c.Check(got, DeepEquals, expect)
}

package notif_test

import (
	"encoding/json"
	"errors"

	"github.com/synapse-garden/sg-proto/notif"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream/river"

	. "gopkg.in/check.v1"
)

type testRiver struct {
	sends [][]byte
}

func (t *testRiver) Send(bs []byte) error {
	t.sends = append(t.sends, bs)
	return nil
}

func (*testRiver) Close() error { return nil }

var _ = river.Pub(&testRiver{})

type testResourcer struct {
	X int `json:"x"`
}

func (testResourcer) Resource() store.Resource {
	return store.Resource("test")
}

type badResourcer map[int]int

func (badResourcer) Resource() store.Resource { return store.Resource("bad") }

func (badResourcer) MarshalJSON() ([]byte, error) { return nil, errors.New("oops") }

func (s *NotifSuite) TestDefaultEncoderEncode(c *C) {
	t := &testRiver{}
	top := notif.MakeUserTopic("bob")
	v := testResourcer{}

	err := notif.DefaultEncoder.Encode(t, v, top)
	c.Assert(err, IsNil)

	vbs, err := json.Marshal(v)
	c.Assert(err, IsNil)
	jsbs, err := json.Marshal(&store.ResourceBox{
		Name:     v.Resource(),
		Contents: vbs,
	})
	c.Assert(err, IsNil)

	c.Check(t.sends[0], DeepEquals, append(top.Code(), jsbs...))
}

func (s *NotifSuite) TestEncode(c *C) {
	t := &testRiver{}
	top := notif.MakeUserTopic("bob")
	v := testResourcer{}

	c.Assert(notif.DefaultEncoder.Encode(t, v, top), IsNil)
	c.Assert(notif.Encode(t, v, top), IsNil)

	c.Check(t.sends[0], DeepEquals, t.sends[1])

	c.Check(notif.Encode(t, badResourcer{}, top), ErrorMatches, "json: .*oops")
}

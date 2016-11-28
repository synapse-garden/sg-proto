package ws_test

import (
	"encoding/json"
	"errors"

	"github.com/synapse-garden/sg-proto/rest/ws"

	. "gopkg.in/check.v1"
)

func (s *WSSuite) TestMessageErrorMarshalJSON(c *C) {
	bs, err := json.Marshal(ws.MessageError{errors.New("oops")})
	c.Assert(err, IsNil)
	c.Check(string(bs), Equals, `{"error":"oops"}`)
}

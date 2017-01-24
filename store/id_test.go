package store_test

import (
	"bytes"
	"encoding/json"

	"github.com/synapse-garden/sg-proto/store"

	uuid "github.com/satori/go.uuid"
	. "gopkg.in/check.v1"
)

var _ = json.Marshaler(store.ID(uuid.Nil))
var _ = json.Unmarshaler(new(store.ID))

func (s *StoreSuite) TestIDJSON(c *C) {
	id := uuid.NewV4()

	bs, err := json.Marshal(store.ID(id))
	c.Assert(err, IsNil)

	into := new(store.ID)
	c.Assert(json.Unmarshal(bs, into), IsNil)
	c.Check(bytes.Equal(id[:], into[:]), Equals, true)
}

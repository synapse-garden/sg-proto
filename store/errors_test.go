package store_test

import (
	"fmt"

	"github.com/synapse-garden/sg-proto/store"

	. "gopkg.in/check.v1"
)

func (s *StoreSuite) TestKeyError(c *C) {
	k, b := []byte("key"), []byte("bucket")
	e := &store.ExistsError{
		Key:    k,
		Bucket: b,
	}
	m := &store.MissingError{
		Key:    k,
		Bucket: b,
	}

	c.Check(e, ErrorMatches, fmt.Sprintf("key %#q already exists in bucket %#q", k, b))
	c.Check(m, ErrorMatches, fmt.Sprintf("key %#q not in bucket %#q", k, b))
}

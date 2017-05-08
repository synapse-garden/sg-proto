package client

import (
	"github.com/synapse-garden/sg-proto/stream"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/pkg/errors"
)

// NewStream POSTs a new Stream to the given Client, with the given
// Name, owned by "from", with "from" and "to" both having read and
// write access.  It returns the new Stream or any error.
func NewStream(c *Client, name, from, to string) (*stream.Stream, error) {
	// Create Stream if not exist
	str := &stream.Stream{
		Group: users.Group{
			Owner: from,
			Readers: map[string]bool{
				from: true,
				to:   true,
			},
			Writers: map[string]bool{
				from: true,
				to:   true,
			},
		},

		Name: name,
	}
	if err := c.CreateStream(str); err != nil {
		return nil, errors.Wrap(err, "failed to create Stream")
	}
	return str, nil
}

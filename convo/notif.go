package convo

import (
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream"
)

// Connected is a Resourcer which can notify that a user has connected.
type Connected stream.ConnectionNotif

// Resource implements Resourcer.Resource on Connected.
func (Connected) Resource() store.Resource { return "convo-connected" }

// Disconnected is a Resourcer which can notify that a user has
// disconnected.
type Disconnected stream.ConnectionNotif

// Resource implements Resourcer.Resource on Disconnected.
func (Disconnected) Resource() store.Resource { return "convo-disconnected" }

// Connected returns a store.Resourcer which can notify that a user
// has connected.
func (c *Convo) Connected(user string) store.Resourcer {
	return Connected{
		UserID:   user,
		StreamID: c.ID,
	}
}

// Disconnected returns a store.Resourcer which can notify that a user
// has disconnected.
func (c *Convo) Disconnected(user string) store.Resourcer {
	return Disconnected{
		UserID:   user,
		StreamID: c.ID,
	}
}

// Deleteed is a Resourcer which can notify that the convo has been
// deleted.
type Deleted string

// Resource implements Resourcer.Resource on Deleted.
func (Deleted) Resource() store.Resource { return "convo-deleted" }

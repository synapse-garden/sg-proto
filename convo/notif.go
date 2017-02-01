package convo

import "github.com/synapse-garden/sg-proto/store"

// ConnectionNotif is a base for convo Resourcers to create notifs.
// Implement store.Resourcer as a method on an alias of ConnectionNotif.
type ConnectionNotif struct {
	UserID  string `json:"userID"`
	ConvoID string `json:"convoID"`
}

// Connected is a ConnectionNotif for convo connection events
type Connected ConnectionNotif

// Resource implements Resourcer.Resource on Connected.
func (Connected) Resource() store.Resource { return "convo-connected" }

// Disconnected is a Resourcer which can notify that a user has
// disconnected.
type Disconnected ConnectionNotif

// Resource implements Resourcer.Resource on Disconnected.
func (Disconnected) Resource() store.Resource { return "convo-disconnected" }

// Connected returns a store.Resourcer which can notify that a user
// has connected.
func (c *Convo) Connected(user string) store.Resourcer {
	return Connected{
		UserID:  user,
		ConvoID: c.ID,
	}
}

// Disconnected returns a store.Resourcer which can notify that a user
// has disconnected.
func (c *Convo) Disconnected(user string) store.Resourcer {
	return Disconnected{
		UserID:  user,
		ConvoID: c.ID,
	}
}

// Deleted is a Resourcer which can notify that the convo has been
// deleted.
type Deleted string

// Resource implements Resourcer.Resource on Deleted.
func (Deleted) Resource() store.Resource { return "convo-deleted" }

// Removed is a Resourcer which can notify that the user has been
// removed from the Convo without showing the user the Convo.
type Removed string

// Resource implements Resourcer.Resource on Removed.
func (Removed) Resource() store.Resource { return "convo-removed" }

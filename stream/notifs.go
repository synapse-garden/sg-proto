package stream

import "github.com/synapse-garden/sg-proto/store"

// Removed is a notification Resourcer that can inform a user they have
// been removed from the Stream without informing them of any other
// information about the Stream.
type Removed string

// Resource implements Resourcer.Resource on Removed.
func (Removed) Resource() store.Resource { return "stream-removed" }

// ConnectionNotif is a base for stream Resourcers to create notifs.
// Implement store.Resourcer as a method on an alias of ConnectionNotif.
type ConnectionNotif struct {
	UserID   string `json:"userID"`
	StreamID string `json:"streamID"`
}

// Connected is a notification Resourcer that can inform a user someone
// has joined the Stream.
type Connected ConnectionNotif

// Resource implements Resourcer.Resource on Connected.
func (Connected) Resource() store.Resource { return "stream-connected" }

// Disconnected is a notification Resourcer that can inform a user
// someone has left the Stream.
type Disconnected ConnectionNotif

// Resource implements Resourcer.Resource on Disconnected.
func (Disconnected) Resource() store.Resource { return "stream-disconnected" }

// Deleted is a notification Resourcer that notifies the user a resource
// has been deleted.
type Deleted string

// Resource implements Resourcer.Resource on Deleted.
func (Deleted) Resource() store.Resource { return "stream-deleted" }

// Connected is a method on Stream which returns a Resourcer for the
// connection notif.
func (s *Stream) Connected(user string) store.Resourcer {
	return Connected{
		StreamID: s.ID,
		UserID:   user,
	}
}

// Disconnected is a method on Stream which returns a Resourcer for the
// disconnection notif.
func (s *Stream) Disconnected(user string) store.Resourcer {
	return Disconnected{
		StreamID: s.ID,
		UserID:   user,
	}
}

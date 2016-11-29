package stream

import "github.com/synapse-garden/sg-proto/store"

// Removed is a notification Resourcer that can inform a user they have
// been removed from the Stream without informing them of any other
// information about the Stream.
type Removed string

// Resource implements Resourcer.Resource on Removed.
func (Removed) Resource() store.Resource { return "removed" }

// Connected is a notification Resourcer that can inform a user someone
// has joined the Stream.
type Connected string

// Resource implements Resourcer.Resource on Connected.
func (Connected) Resource() store.Resource { return "connected" }

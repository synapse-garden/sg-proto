package task

import "github.com/synapse-garden/sg-proto/store"

func (*Task) Resource() store.Resource { return "tasks" }

// Deleted is a Resourcer which can notify that the convo has been
// deleted.
type Deleted string

// Resource implements Resourcer.Resource on Deleted.
func (Deleted) Resource() store.Resource { return "task-deleted" }

// Removed is a Resourcer which can be used to notify that the user has
// been removed from the Task without showing them the Task itself.
type Removed ID

// Resource implements Resourcer on Removed.
func (Removed) Resource() store.Resource { return "task-removed" }

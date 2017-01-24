package task

import "github.com/synapse-garden/sg-proto/store"

func (*Task) Resource() store.Resource { return "tasks" }

// Deleted is a Resourcer which can notify that the convo has been
// deleted.
type Deleted string

// Resource implements Resourcer.Resource on Deleted.
func (Deleted) Resource() store.Resource { return "task-deleted" }

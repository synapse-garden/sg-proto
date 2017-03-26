package task_test

import (
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/task"
)

var (
	_ = store.Resourcer(new(task.Task))
	_ = store.Resourcer(task.Deleted("hi"))
)

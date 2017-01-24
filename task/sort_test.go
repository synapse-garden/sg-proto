package task_test

import (
	"sort"
	"time"

	"github.com/synapse-garden/sg-proto/task"

	. "gopkg.in/check.v1"
)

var _ = sort.Interface(task.ByOldest{})

func (s *TaskSuite) TestSort(c *C) {
	var (
		now  = time.Now()
		late = now.Add(-1 * time.Hour)
		soon = now.Add(1 * time.Hour)

		sorted = task.ByOldest{
			&task.Task{Due: &now},
			&task.Task{},
			&task.Task{Due: &soon},
			&task.Task{Due: &late},
		}
	)

	sort.Sort(sorted)

	c.Check(sorted, DeepEquals, task.ByOldest{
		&task.Task{Due: &late},
		&task.Task{Due: &now},
		&task.Task{Due: &soon},
		&task.Task{},
	})
}

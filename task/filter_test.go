package task_test

import (
	"time"

	"github.com/synapse-garden/sg-proto/task"

	. "gopkg.in/check.v1"
)

func (s *TaskSuite) TestFilters(c *C) {
	someWhen := time.Now()
	beforeNow := someWhen.Add(-1 * time.Hour)
	afterNow := someWhen.Add(1 * time.Hour)
	oneDayLater := someWhen.Add(24 * time.Hour)

	for i, test := range []struct {
		should string
		given  *task.Task
		filter task.Filter
		expect bool
	}{{
		should: "not be a member for a complete task",
		given:  &task.Task{Completed: true},
		filter: task.Incomplete,
		expect: false,
	}, {
		should: "be a member for an incomplete task",
		given:  new(task.Task),
		filter: task.Incomplete,
		expect: true,
	}, {
		should: "not be a member for an incomplete task",
		given:  new(task.Task),
		filter: task.Complete,
		expect: false,
	}, {
		should: "be a member for a complete task",
		given:  &task.Task{Completed: true},
		filter: task.Complete,
		expect: true,
	}, {
		should: "not be a member for wrong constant",
		given:  &task.Task{Completed: true},
		filter: task.Completion(5),
		expect: false,
	}, {
		should: "return true for overdue tasks",
		given:  &task.Task{Due: &beforeNow},
		filter: task.Overdue(someWhen),
		expect: true,
	}, {
		should: "return false for overdue complete tasks",
		given:  &task.Task{Completed: true, Due: &beforeNow},
		filter: task.Overdue(someWhen),
		expect: false,
	}, {
		should: "return false for not-overdue tasks",
		given:  &task.Task{Due: &afterNow},
		filter: task.Overdue(someWhen),
		expect: false,
	}, {
		should: "return false for overdue tasks",
		given:  &task.Task{Due: &beforeNow},
		filter: task.NotYetDue(someWhen),
		expect: false,
	}, {
		should: "return true for not-overdue tasks",
		given:  &task.Task{Due: &afterNow},
		filter: task.NotYetDue(someWhen),
		expect: true,
	}, {
		should: "return true for tasks within window",
		given:  &task.Task{Due: &afterNow},
		filter: task.DueWithin{
			From: beforeNow,
			Til:  oneDayLater,
		},
		expect: true,
	}, {
		should: "return false for tasks after window",
		given:  &task.Task{Due: &oneDayLater},
		filter: task.DueWithin{
			From: beforeNow,
			Til:  afterNow,
		},
		expect: false,
	}, {
		should: "return false for tasks before window",
		given:  &task.Task{Due: &beforeNow},
		filter: task.DueWithin{
			From: someWhen,
			Til:  afterNow,
		},
		expect: false,
	}, {
		should: "return false for tasks with no due date",
		given:  new(task.Task),
		filter: task.DueWithin{
			From: beforeNow,
			Til:  oneDayLater,
		},
		expect: false,
	}} {
		c.Logf("test %d: should %s", i, test.should)
		c.Check(test.filter.Member(test.given), Equals, test.expect)
	}
}

package testing_test

import (
	"time"

	"github.com/synapse-garden/sg-proto/testing"

	. "gopkg.in/check.v1"
)

func (s *TestingSuite) TestTimerNow(c *C) {
	now := time.Now()
	t := testing.Timer(now)
	c.Check(t.Now(), Equals, now)
}

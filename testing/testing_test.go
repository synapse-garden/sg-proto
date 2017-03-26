package testing_test

import (
	tt "testing"

	. "gopkg.in/check.v1"
)

func Test(t *tt.T) { TestingT(t) }

type TestingSuite struct{}

var _ = Suite(&TestingSuite{})

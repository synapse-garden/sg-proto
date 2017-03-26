package admin_test

import (
	"testing"

	"github.com/synapse-garden/sg-proto/admin"
	"github.com/synapse-garden/sg-proto/auth"

	uuid "github.com/satori/go.uuid"
	. "gopkg.in/check.v1"
)

type AdminSuite struct{}

var _ = Suite(new(AdminSuite))

func Test(t *testing.T) { TestingT(t) }

func (s *AdminSuite) TestErrNotFound(c *C) {
	uu := uuid.NewV4()
	tok := auth.Token(uu[:])
	e := admin.ErrNotFound(tok)
	c.Check(e.Error(), Equals, "no such admin token `"+tok.String()+"`")
	c.Check(admin.IsNotFound(e), Equals, true)
}

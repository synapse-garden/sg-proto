package users_test

import (
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/users"
)

var _ = store.Resourcer(&users.User{})

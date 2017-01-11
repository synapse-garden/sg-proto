package users_test

import "github.com/synapse-garden/sg-proto/users"

var (
	_ = users.Filter(users.ByOwner(""))
	_ = users.Filter(users.ByReader(""))
	_ = users.Filter(users.MultiOr(nil))
	_ = users.Filter(users.MultiAnd(nil))
)

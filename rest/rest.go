package rest

import (
	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	"github.com/julienschmidt/httprouter"
)

// Needed endpoints:
//
// Create a new user:
//  - POST /incept :credential (returns user_id or error)
//  - POST /incept/:credential ("magic link")
//
// Get a new session / login:
//  - POST /session/:user_id :pwhash (returns session key)
//  - GET  /profile (user ID inferred)
//
// Open a new chat socket
//  - GET /chat/:user_id
//
// TODOs / tasks
//  - POST /todo {bounty, due}
//  - POST /todo/:id/complete => Get bounty if before due

// API binds some functions on an httprouter.Router.
type API func(*httprouter.Router, *bolt.DB)

// Bind binds the API on the given DB.  It sets up REST endpoints as needed.
func Bind(db *bolt.DB) (*httprouter.Router, error) {
	if err := db.Update(store.Prep(
		incept.TicketBucket,
		users.UserBucket,
		auth.LoginBucket,
	)); err != nil {
		return nil, err
	}
	htr := httprouter.New()
	for _, api := range []API{
		Incept,
	} {
		api(htr, db)
	}

	return htr, nil
}

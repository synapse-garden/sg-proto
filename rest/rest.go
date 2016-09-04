package rest

import (
	"github.com/synapse-garden/sg-proto/admin"
	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	"github.com/julienschmidt/httprouter"
)

// Needed endpoints:
//
// Admin:
//  - Admin auth middleware??  Or always send admin account info?  Or
//    simply have admin API key from CLI for now?
//  - POST /admin/tickets
//  - DELETE /admin/tickets/:credential
//
// Create a new user:
//  - POST /incept/:credential ("magic link")
//
// Get a new session / login:
//  - POST /session/:user_id :pwhash (returns Token to be included as Authorization: Bearer)
//
// User account
//  - GET  /profile (user ID inferred) => /users/:id
//  - PUT  /profile
//  - DELETE /profile => delete user account and any logins
//
// Open a new chat socket
//  - GET /chat/:user_id
//
// TODOs / tasks
//  - POST /todo {bounty, due}
//  - POST /todo/:id/complete => Get bounty if before due

// API is a transform on an httprouter.Router, passing a DB for passing
// on to httprouter.Handles.
type API func(*httprouter.Router, *bolt.DB) error

// Bind binds the API on the given DB.  It sets up REST endpoints as needed.
func Bind(
	db *bolt.DB,
	source *SourceInfo,
	apiKey auth.Token,
) (*httprouter.Router, error) {
	if err := db.Update(store.Prep(
		admin.AdminBucket,
		incept.TicketBucket,
		users.UserBucket,
		auth.LoginBucket,
		auth.SessionBucket,
		auth.RefreshBucket,
		auth.ContextBucket,
	)); err != nil {
		return nil, err
	}
	htr := httprouter.New()
	for _, api := range []API{
		Source(source),
		Admin(apiKey),
		Incept,
		Token,
		Profile,
	} {
		if err := api(htr, db); err != nil {
			return nil, err
		}
	}

	return htr, nil
}

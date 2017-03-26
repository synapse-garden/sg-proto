package rest

import (
	"github.com/synapse-garden/sg-proto/admin"
	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/convo"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream"
	"github.com/synapse-garden/sg-proto/stream/river"
	"github.com/synapse-garden/sg-proto/task"
	"github.com/synapse-garden/sg-proto/text"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	"github.com/julienschmidt/httprouter"
)

// Needed endpoints:
//
// Admin:
//  - Admin auth middleware
//  - POST /admin/tickets (optionally ?count=n)
//  - GET /admin/tickets
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
//  - PUT  /profile (nothing to update yet)
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
type API interface {
	Bind(*httprouter.Router) error
}

// Bind binds the API on the given DB.  It sets up REST endpoints as needed.
func Bind(
	db *bolt.DB,
	source SourceInfo,
	apiKey auth.Token,
) (*httprouter.Router, error) {
	if err := db.Update(store.Wrap(
		store.Prep(
			admin.AdminBucket,
			incept.TicketBucket,
			users.UserBucket,
			auth.LoginBucket,
			auth.SessionBucket,
			auth.RefreshBucket,
			auth.ContextBucket,
			stream.StreamBucket,
			river.RiverBucket,
			convo.ConvoBucket,
			convo.MessageBucket,
			text.TextBucket,
			task.TaskBucket,
		),
		auth.ClearSessions,
		river.ClearRivers,
	)); err != nil {
		return nil, err
	}

	htr := httprouter.New()
	for _, api := range []API{
		source,
		Incept{DB: db},
		Token{DB: db},
		Profile{DB: db},
		// Note that notifying APIs must be references since the
		// notif connect sets a Pub socket handle in the struct.
		&Stream{DB: db},
		&Convo{DB: db},
		&Task{DB: db},
		&Admin{Token: apiKey, DB: db},
		// Connect Notif last so Pubs are already registered.
		Notif{DB: db},
	} {
		if err := api.Bind(htr); err != nil {
			return nil, err
		}
	}

	return htr, nil
}

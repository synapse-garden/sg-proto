package rest

import (
	"github.com/synapse-garden/sg-proto/store"

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

type HTTPError struct {
	cause   error
	message []byte
}

func (h *HTTPError) Error() string { return h.cause.Error() }
func (h *HTTPError) Read() ([]byte, error) {
	return h.message, nil
}

// API binds some functions on an httprouter.Router.
type API func(*httprouter.Router, *bolt.DB)

// Bind binds the API on the given DB.  It sets up REST endpoints as needed.
func Bind(db *bolt.DB) (*httprouter.Router, error) {
	if err := db.Update(store.Prep); err != nil {
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

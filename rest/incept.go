package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Incept implements API on a database.  It handles new user creation.
type Incept struct {
	*bolt.DB
}

// Bind implements API.Bind on Incept.
func (i Incept) Bind(r *httprouter.Router) (Cleanup, error) {
	if i.DB == nil {
		return nil, errors.New("nil Incept DB handle")
	}
	r.POST("/incept/:key", i.Incept)
	return nil, nil
}

func (i Incept) Incept(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	l := new(auth.Login)
	if err := json.NewDecoder(r.Body).Decode(l); err != nil {
		http.Error(w, fmt.Sprintf(
			"failed to decode: %s",
			err.Error(),
		), http.StatusBadRequest)
		return
	}

	if err := auth.ValidateNew(l); err != nil {
		http.Error(w, fmt.Sprintf(
			"invalid login: %s",
			err.Error(),
		), http.StatusBadRequest)
		return
	}

	key := ps.ByName("key")
	tkt, err := uuid.FromString(key)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	err = incept.Incept(incept.Ticket(tkt), l, i.DB)
	if err != nil {
		var status int
		switch err.(type) {
		case incept.ErrTicketMissing:
			status = http.StatusNotFound
		case users.ErrExists:
			status = http.StatusConflict
		case auth.ErrExists:
			status = http.StatusConflict
		default:
			status = http.StatusInternalServerError
		}
		http.Error(w, err.Error(), status)
		return
	}

	err = json.NewEncoder(w).Encode(l.User)
	if err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to marshal User").Error(),
			http.StatusInternalServerError)
		return
	}
}

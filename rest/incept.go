package rest

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	"github.com/julienschmidt/httprouter"
	uuid "github.com/satori/go.uuid"
)

func Incept(r *httprouter.Router, db *bolt.DB) {
	r.POST("/incept/:key", InceptHandle(db))
}

func InceptHandle(db *bolt.DB) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		u := new(users.User)
		if err := json.NewDecoder(r.Body).Decode(u); err != nil {
			http.Error(w, fmt.Sprintf(
				"failed to decode: %s",
				err.Error(),
			), http.StatusBadRequest)
			return
		}

		if err := users.ValidateNew(u); err != nil {
			http.Error(w, fmt.Sprintf(
				"invalid user: %s",
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

		err = incept.Incept(w, incept.Ticket(tkt), u, db)
		switch {
		case store.IsMissing(err):
			http.Error(w, fmt.Sprintf(
				"no such ticket %q",
				key,
			), http.StatusBadRequest)
			return
		case store.IsExists(err):
			http.Error(w, fmt.Sprintf(
				"user %q already exists",
				u.Name,
			), http.StatusBadRequest)
			return
		case err != nil:
			http.Error(
				w,
				err.Error(),
				http.StatusInternalServerError,
			)
			return
		}
	}
}

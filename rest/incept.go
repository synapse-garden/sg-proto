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
	uuid "github.com/satori/go.uuid"
)

func Incept(r *httprouter.Router, db *bolt.DB) {
	r.POST("/incept/:key", InceptHandle(db))
}

func InceptHandle(db *bolt.DB) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
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

		err = incept.Incept(w, incept.Ticket(tkt), l, db)
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
	}
}

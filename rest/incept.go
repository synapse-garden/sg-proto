package rest

import (
	"encoding/json"
	"net/http"

	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/user"

	"github.com/boltdb/bolt"
	"github.com/julienschmidt/httprouter"
)

func Incept(r *httprouter.Router, db *bolt.DB) {
	r.POST("/incept/:key", InceptHandle(db))
}

func InceptHandle(db *bolt.DB) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		u := new(user.User)
		err := json.NewDecoder(r.Body).Decode(u)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = incept.Incept(
			w,
			incept.Ticket(ps.ByName("key")),
			u,
			db,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
	}
}

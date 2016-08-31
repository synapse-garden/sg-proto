package rest

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/synapse-garden/sg-proto/auth"

	"github.com/boltdb/bolt"
	htr "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

func Token(r *htr.Router, db *bolt.DB) error {
	r.POST("/tokens", HandleToken(db))

	return nil
}

func HandleToken(db *bolt.DB) htr.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ htr.Params) {
		// Unmarshal the Login from the Body
		l := new(auth.Login)
		if err := json.NewDecoder(r.Body).Decode(l); err != nil {
			http.Error(w, errors.Wrap(
				err, "failed to decode Login").Error(),
				http.StatusBadRequest)
			return
		}

		if err := db.View(auth.Check(l)); err != nil {
			switch err.(type) {
			case auth.ErrInvalid, auth.ErrMissing:
				http.Error(w, err.Error(), http.StatusNotFound)
			default:
				http.Error(w, errors.Wrap(
					err, "failed to compare logins",
				).Error(), http.StatusInternalServerError)
			}
			return
		}

		sesh := &auth.Session{}
		if err := db.Update(auth.NewSession(
			sesh,
			time.Now().Add(auth.Expiration),
			auth.Expiration,
			auth.NewToken(auth.BearerType),
			auth.NewToken(auth.RefreshType),
		)); err != nil {
			http.Error(w, errors.Wrap(
				err, "failed to create new session",
			).Error(), http.StatusInternalServerError)

			return
		}

		if err := json.NewEncoder(w).Encode(sesh); err != nil {
			log.Printf("failed to create new session %#v: %#v", sesh, err)
			http.Error(w, errors.Wrap(
				err, "failed to create new session",
			).Error(), http.StatusInternalServerError)

			return
		}
	}
}
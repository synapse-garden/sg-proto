package rest

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/synapse-garden/sg-proto/auth"

	"github.com/boltdb/bolt"
	htr "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

func Token(r *htr.Router, db *bolt.DB) error {
	r.POST("/tokens", HandleToken(db))
	// TODO: r.DELETE("/tokens", HandleLogoutUser(db))
	r.DELETE("/tokens/:token", HandleDeleteToken(db))

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
			l.Name,
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

func HandleDeleteToken(db *bolt.DB) htr.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps htr.Params) {
		escaped := ps.ByName("token")
		unescaped, err := url.QueryUnescape(escaped)
		if err != nil {
			http.Error(w, errors.Wrapf(
				err, "token not URL-escaped", escaped,
			).Error(), http.StatusBadRequest)
			return
		}
		token, err := auth.DecodeToken(unescaped)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// len == 0 case handled by other handler
		if err = db.View(auth.CheckToken(token)); err != nil {
			var code int
			switch err.(type) {
			case auth.ErrMissingSession, auth.ErrTokenExpired:
				code = http.StatusNotFound
			default:
				code = http.StatusInternalServerError
			}

			http.Error(w, err.Error(), code)
			return
		}

		if err := db.Update(auth.DeleteToken(token)); err != nil {
			var code int
			switch err.(type) {
			case auth.ErrMissingSession:
				code = http.StatusNotFound
			default:
				// Something weird happened.  Maybe you already logged out.
				code = http.StatusInternalServerError
				log.Printf("Token: %#v\nCheckToken "+
					"passed but DeleteToken "+
					"failed: %#v", token, err)
			}

			http.Error(w, err.Error(), code)
			return
		}
	}
}

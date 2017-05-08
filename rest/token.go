package rest

import (
	"encoding/json"
	"log"
	"net/http"
	"time"

	"github.com/synapse-garden/sg-proto/auth"
	mw "github.com/synapse-garden/sg-proto/rest/middleware"

	"github.com/boltdb/bolt"
	htr "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// Token implements API.  It handles creating and deleting login Tokens.
type Token struct{ *bolt.DB }

// Bind implements API.Bind on Token.
func (t Token) Bind(r *htr.Router) (Cleanup, error) {
	if t.DB == nil {
		return nil, errors.New("nil Token DB handle")
	}
	r.POST("/tokens", t.Create)
	r.DELETE("/tokens", mw.AuthUser(t.Delete, t.DB, mw.CtxSetToken))

	return nil, nil
}

func (t Token) Create(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	// Unmarshal the Login from the Body
	l := new(auth.Login)
	if err := json.NewDecoder(r.Body).Decode(l); err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to decode Login").Error(),
			http.StatusBadRequest)
		return
	}

	if err := t.View(auth.Check(l)); err != nil {
		switch err.(type) {
		case auth.ErrInvalid, auth.ErrMissing:
			http.Error(w, err.Error(), http.StatusNotFound)
		case auth.ErrDisabled:
			http.Error(w, err.Error(), http.StatusUnauthorized)
		default:
			http.Error(w, errors.Wrap(
				err, "failed to compare logins",
			).Error(), http.StatusInternalServerError)
		}
		return
	}

	sesh := &auth.Session{}
	if err := t.Update(auth.NewSession(
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

func (t Token) Delete(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	token := mw.CtxGetToken(r)

	// len == 0 case handled by other handler
	if err := t.View(auth.CheckToken(token)); err != nil {
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

	if err := t.Update(auth.DeleteToken(token)); err != nil {
		var code int
		switch err.(type) {
		case auth.ErrMissingSession:
			code = http.StatusNotFound
		default:
			// Something weird happened.  Maybe they already
			// logged out.
			code = http.StatusInternalServerError
			log.Printf("Token: %#v\nCheckToken passed but "+
				"DeleteToken failed: %#v", token, err)
		}

		http.Error(w, err.Error(), code)
		return
	}
}

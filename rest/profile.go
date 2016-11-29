package rest

import (
	"encoding/json"
	"net/http"

	"github.com/synapse-garden/sg-proto/auth"
	mw "github.com/synapse-garden/sg-proto/rest/middleware"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	htr "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

// Profile implements API.  It handles user profiles.
type Profile struct {
	*bolt.DB
}

// Bind implements API.Bind on Profile.
func (p Profile) Bind(r *htr.Router) error {
	db := p.DB
	if db == nil {
		return errors.New("Profile DB handle must not be nil")
	}
	r.GET("/profile", mw.AuthUser(p.Get, db, mw.CtxSetUserID))
	r.DELETE("/profile", mw.AuthUser(p.Delete, db, mw.ByFields(
		auth.CtxUserID,
		auth.CtxToken,
		auth.CtxRefreshToken,
	)))

	return nil
}

func (p Profile) Get(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	userID := mw.CtxGetUserID(r)
	user := new(users.User)
	err := p.View(store.Unmarshal(users.UserBucket, user, []byte(userID)))
	if err != nil {
		switch err.(type) {
		case users.ErrMissing:
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		default:
			http.Error(w, errors.Wrapf(err, "unexpected error retrieving user %#q", userID).Error(), http.StatusInternalServerError)
			return
		}
	}

	if err := json.NewEncoder(w).Encode(user); err != nil {
		http.Error(w, errors.Wrap(err, "failed to unmarshal user").Error(), http.StatusInternalServerError)
		return
	}
}

func (p Profile) Delete(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	userID := mw.CtxGetUserID(r)
	token := mw.CtxGetToken(r)
	refreshToken := mw.CtxGetRefreshToken(r)

	err := p.Update(store.Wrap(
		users.Delete(&users.User{Name: userID}),
		auth.Delete(&auth.Login{
			User: users.User{Name: userID},
		}),
		auth.DeleteSession(&auth.Session{
			Token:        token,
			RefreshToken: refreshToken,
		}),
	))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

package middleware

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/store"

	"github.com/boltdb/bolt"
	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
)

type Header string

const (
	AuthHeader    Header = "Authorization"
	RefreshHeader Header = "X-Auth-Refresh"
)

func GetToken(kind, from string) ([]byte, error) {
	// Token is expected to be base64 encoded byte slice.
	// Kind is assumed to be valid.
	substrings := strings.SplitN(from, kind+" ", 2)
	switch {
	case len(from) == 0:
		return nil, errors.Errorf(
			"no %q token provided in header %q",
			kind, AuthHeader)
	case len(substrings) != 2:
		return nil, errors.Errorf(
			"invalid %q token provided in header %q",
			kind, AuthHeader)
	case substrings[0] != "":
		return nil, errors.Errorf(
			"invalid %q token kind, use %q",
			AuthHeader, kind)
	}

	bs, err := base64.StdEncoding.DecodeString(substrings[1])
	if err != nil {
		return nil, errors.Wrapf(err,
			"failed to decode %q token", kind)
	}

	return auth.Token(bs), nil
}

func Auth(h httprouter.Handle, db *bolt.DB) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Is an authorized key in the header?
		token, err := GetToken(
			"Bearer",
			r.Header.Get(string(AuthHeader)),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// TODO: Split me into my own function.
		// Check whether the session token is valid
		err = db.View(auth.CheckToken(token))
		if err == nil {
			h(w, r, ps)
			return
		}

		switch err.(type) {
		case auth.ErrMissingSession:
			http.Error(w, "invalid session token", http.StatusUnauthorized)
			return
		case auth.ErrTokenExpired:
			rToken, err := GetToken(
				"Refresh",
				r.Header.Get(string(RefreshHeader)),
			)
			if err != nil {
				http.Error(w, errors.Wrap(err, "invalid refresh token").Error(), http.StatusUnauthorized)
				return
			}
			if err := db.View(auth.CheckRefresh(rToken)); store.IsMissing(err) {
				http.Error(w, errors.Wrap(err, "invalid refresh token").Error(), http.StatusUnauthorized)
				return
			} else if err != nil {
				http.Error(w, errors.Wrap(err, "failed to check refresh token").Error(), http.StatusInternalServerError)
				log.Printf("Failed to check refresh token: %#v", err)
				return
			}
			sess := &auth.Session{Token: token, RefreshToken: rToken}
			err = db.Update(auth.RefreshIfValid(sess, time.Now().Add(auth.Expiration), auth.Expiration))
			if err != nil {
				switch err.(type) {
				case auth.ErrMissingSession:
					http.Error(w, "invalid refresh token", http.StatusUnauthorized)
				default:
					http.Error(w, errors.Wrap(err, "failed to refresh session").Error(), http.StatusInternalServerError)
					log.Printf("Failed to refresh session: %#v", err)
				}
				return
			}

			if err := db.View(auth.CheckToken(token)); err != nil {
				switch err.(type) {
				case auth.ErrMissingSession:
					http.Error(w, "invalid session token", http.StatusUnauthorized)
					// Something weird happened.
					log.Printf("Failed to verify session token after refresh: %#v", err)
					return
				default:
					http.Error(w, errors.Wrap(err, "failed to verify session token after refresh").Error(), http.StatusInternalServerError)
					log.Printf("Failed to verify session token after refresh: %#v", err)
					return
				}
			}
			// If we ran the gamut of possible errors here, we're in the clear and the session was refreshed.  Carry on.
		default:
			http.Error(w, errors.Wrap(err, "unexpected server error").Error(), http.StatusInternalServerError)
			return
		}

	}
}

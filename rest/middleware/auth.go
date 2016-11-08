package middleware

import (
	"encoding/base64"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/synapse-garden/sg-proto/admin"
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

	WSProtocolsHeader Header = "Sec-WebSocket-Protocol"
)

func GetToken(kind auth.TokenType, from string) ([]byte, error) {
	// Token is expected to be base64 encoded byte slice.
	// Kind is assumed to be valid.
	substrings := strings.SplitN(from, kind.String()+" ", 2)
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

	return auth.DecodeToken(substrings[1])
}

// GetWSToken looks for an Authorization token of the given kind in the
// comma-separated list of websocket subprotocols.  It expects the token
// as an unpadded base64url-encoded string, prepended by '{kind}+'.
func GetWSToken(protocols string, kind auth.TokenType) ([]byte, error) {
	for _, pro := range strings.Split(protocols, ",") {
		if len(pro) == 0 {
			continue
		}
		substrings := strings.SplitN(pro, kind.String()+"+", 2)
		switch {
		case len(substrings) != 2:
			fallthrough
		case substrings[0] != "":
			continue
		}

		return base64.RawURLEncoding.DecodeString(substrings[1])
	}

	return nil, errors.Errorf("no %q token found in %#q",
		kind.String(),
		protocols,
	)
}

func AuthUser(h httprouter.Handle, db *bolt.DB, ctrs ...Contexter) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Is an authorized key in the header?
		token, err := GetToken(
			auth.BearerType,
			r.Header.Get(string(AuthHeader)),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// TODO: Split into my two functions.
		// Check whether the session token is valid
		err = db.View(auth.CheckToken(token))
		switch {
		case err == nil && len(ctrs) == 0:
			h(w, r, ps)
			return
		case err == nil:
			// Apply requested context
			ctx := new(auth.Context)
			if err = db.View(auth.GetContext(ctx, token)); err != nil {
				switch err.(type) {
				case auth.ErrContextMissing:
					// Valid session with no context.
					log.Printf("unexpected error: valid session %#q with no context", token)
					http.Error(w, errors.Wrap(
						err, "error getting session context").Error(),
						http.StatusInternalServerError)
					return
				default:
					http.Error(w, "error getting session context", http.StatusInternalServerError)
					return
				}
			}
			for _, ctr := range ctrs {
				r = ctr(r, ctx)
			}
			h(w, r, ps)
			return
		}

		switch err.(type) {
		case auth.ErrMissingSession:
			http.Error(w, "invalid session token", http.StatusUnauthorized)
			return
		case auth.ErrTokenExpired:
			rToken, err := GetToken(
				auth.RefreshType,
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

func AuthWSUser(h httprouter.Handle, db *bolt.DB, ctrs ...Contexter) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Is an authorized key in the header?
		token, err := GetWSToken(
			r.Header.Get(string(WSProtocolsHeader)),
			auth.BearerType,
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// TODO: Split into my two functions.
		// Check whether the session token is valid
		err = db.View(auth.CheckToken(token))
		switch {
		case err == nil && len(ctrs) == 0:
			h(w, r, ps)
			return
		case err == nil:
			// Apply requested context
			ctx := new(auth.Context)
			if err = db.View(auth.GetContext(ctx, token)); err != nil {
				switch err.(type) {
				case auth.ErrContextMissing:
					// Valid session with no context.
					log.Printf("unexpected error: valid session %#q with no context", token)
					http.Error(w, errors.Wrap(
						err, "error getting session context").Error(),
						http.StatusInternalServerError)
					return
				default:
					http.Error(w, "error getting session context", http.StatusInternalServerError)
					return
				}
			}
			for _, ctr := range ctrs {
				r = ctr(r, ctx)
			}
			h(w, r, ps)
			return
		}

		switch err.(type) {
		case auth.ErrMissingSession:
			http.Error(w, "invalid session token", http.StatusUnauthorized)
			return
		case auth.ErrTokenExpired:
			rToken, err := GetToken(
				auth.RefreshType,
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

func AuthAdmin(h httprouter.Handle, db *bolt.DB) httprouter.Handle {
	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		// Is an authorized key in the header?
		token, err := GetToken(
			auth.AdminType,
			r.Header.Get(string(AuthHeader)),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		err = db.View(admin.CheckToken(token))
		if err != nil {
			switch err.(type) {
			case admin.ErrNotFound:
				http.Error(w, err.Error(), http.StatusUnauthorized)
			default:
				http.Error(w, errors.Wrap(err, "error authorizing admin API key").Error(), http.StatusInternalServerError)
			}
			return
		}

		h(w, r, ps)
	}
}

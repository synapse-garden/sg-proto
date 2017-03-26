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
	fields := strings.Fields(from)
	switch {
	case len(fields) == 0:
		return nil, errors.Errorf("no %q token provided", kind)
	case len(fields) == 1:
		return nil, errors.Errorf("no %q token provided", kind)
	case len(fields) != 2:
		return nil, errors.New("too many token fields")
	case fields[0] != kind.String():
		return nil, errors.Errorf(
			"invalid token kind %q, expected %q",
			fields[0], kind)
	}

	return auth.DecodeToken(fields[1])
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
		bearerToken, err := GetToken(
			auth.BearerType,
			r.Header.Get(string(AuthHeader)),
		)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// TODO: Split into my two functions.
		// Check whether the session token is valid
		err = db.View(auth.CheckToken(bearerToken))
		switch {
		case err == nil && len(ctrs) == 0:
			h(w, r, ps)
			return
		case err == nil:
			// Apply requested context
			ctx := new(auth.Context)
			err = db.View(auth.GetContext(ctx, bearerToken))
			switch {
			case auth.IsContextMissing(err):
				// Valid session with no context.
				log.Printf("unexpected error: valid "+
					"session %#q with no context",
					bearerToken)
				http.Error(w, errors.Wrap(err,
					"error getting session context",
				).Error(), http.StatusInternalServerError)
				return
			case err != nil:
				http.Error(w,
					"error getting session context",
					http.StatusInternalServerError,
				)
				return
			}
			for _, ctr := range ctrs {
				r = ctr(r, ctx)
			}
			h(w, r, ps)
			return
		}

		switch {
		case auth.IsMissingSession(err):
			http.Error(w,
				"invalid session token",
				http.StatusUnauthorized,
			)
			return
		case auth.IsTokenExpired(err):
			rToken, err := GetToken(
				auth.RefreshType,
				r.Header.Get(string(RefreshHeader)),
			)
			if err != nil {
				http.Error(w, errors.Wrap(
					err, "invalid refresh token",
				).Error(), http.StatusUnauthorized)
				return
			}
			err = db.View(auth.CheckRefresh(rToken))
			switch {
			case store.IsMissing(err):
				http.Error(w, errors.Wrap(
					err, "invalid refresh token",
				).Error(), http.StatusUnauthorized)
				return
			case err != nil:
				http.Error(w, errors.Wrap(err,
					"failed to check refresh token",
				).Error(), http.StatusInternalServerError)
				log.Printf("Failed to check refresh "+
					"token: %#v", err)
				return
			}
			sess := &auth.Session{
				Token:        bearerToken,
				RefreshToken: rToken,
			}
			err = db.Update(auth.RefreshIfValid(sess,
				time.Now().Add(auth.Expiration),
				auth.Expiration,
			))
			switch {
			case auth.IsMissingSession(err):
				http.Error(w,
					"invalid refresh token",
					http.StatusUnauthorized,
				)
				return
			case err != nil:
				http.Error(w, errors.Wrap(err,
					"failed to refresh session",
				).Error(), http.StatusInternalServerError)
				log.Printf("Failed to refresh session "+
					": %#v", err)
				return
			}

			err = db.View(auth.CheckToken(bearerToken))
			switch {
			case auth.IsMissingSession(err):
				http.Error(w, "invalid session token",
					http.StatusUnauthorized)
				// Something weird happened.
				log.Printf("Failed to verify session "+
					"token after refresh: %#v", err)
				return
			case err != nil:
				http.Error(w, errors.Wrap(err,
					"failed to verify session "+
						"token after refresh",
				).Error(), http.StatusInternalServerError)
				log.Printf("Failed to verify session "+
					"token after refresh: %#v", err)
				return
			}

			// If we ran the gamut of possible errors here,
			// we're in the clear and the session was
			// refreshed.  Carry on.

			if len(ctrs) == 0 {
				// If there was no context, just apply
				// the handler.
				h(w, r, ps)
				return
			}

			// Otherwise, load the auth context and apply
			// the requested contexts to the request.
			ctx := new(auth.Context)
			err = db.View(auth.GetContext(ctx, bearerToken))
			switch {
			case auth.IsContextMissing(err):
				// Valid session with no context.
				log.Printf("unexpected error: valid "+
					"session %#q with no context",
					bearerToken,
				)
				http.Error(w, errors.Wrap(
					err, "error getting session context",
				).Error(), http.StatusInternalServerError)
				return
			case err != nil:
				http.Error(w,
					"error getting session context",
					http.StatusInternalServerError,
				)
				return
			}
			for _, ctr := range ctrs {
				r = ctr(r, ctx)
			}
			h(w, r, ps)
			return
		default:
			http.Error(w, errors.Wrap(
				err, "unexpected server error",
			).Error(), http.StatusInternalServerError)
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
			err = db.View(auth.GetContext(ctx, token))
			switch {
			case auth.IsContextMissing(err):
				// Valid session with no context.
				log.Printf("unexpected error: valid "+
					"session %#q with no context",
					token)
				http.Error(w, errors.Wrap(err,
					"error getting session context",
				).Error(), http.StatusInternalServerError)
				return
			case err != nil:
				http.Error(w,
					"error getting session context",
					http.StatusInternalServerError,
				)
				return
			}
			for _, ctr := range ctrs {
				r = ctr(r, ctx)
			}
			h(w, r, ps)
			return
		}

		switch {
		case auth.IsMissingSession(err):
			http.Error(w, "invalid session token",
				http.StatusUnauthorized)
			return
		case auth.IsTokenExpired(err):
			rToken, err := GetWSToken(
				r.Header.Get(string(WSProtocolsHeader)),
				auth.RefreshType,
			)
			if err != nil {
				http.Error(w, errors.Wrap(err,
					"invalid refresh token",
				).Error(), http.StatusUnauthorized)
				return
			}
			err = db.View(auth.CheckRefresh(rToken))
			switch {
			case store.IsMissing(err):
				http.Error(w, errors.Wrap(
					err, "invalid refresh token",
				).Error(), http.StatusUnauthorized)
				return
			case err != nil:
				http.Error(w, errors.Wrap(err,
					"failed to check refresh token",
				).Error(), http.StatusInternalServerError)
				log.Printf("Failed to check refresh "+
					"token: %#v", err)
				return
			}
			sess := &auth.Session{
				Token:        token,
				RefreshToken: rToken,
			}
			err = db.Update(auth.RefreshIfValid(sess,
				time.Now().Add(auth.Expiration),
				auth.Expiration,
			))
			switch {
			case auth.IsMissingSession(err):
				http.Error(w,
					"invalid refresh token",
					http.StatusUnauthorized,
				)
				return
			case err != nil:
				http.Error(w, errors.Wrap(err,
					"failed to refresh session",
				).Error(), http.StatusInternalServerError)
				log.Printf("Failed to refresh "+
					"session: %#v", err)
				return
			}

			err = db.View(auth.CheckToken(token))
			switch {
			case auth.IsMissingSession(err):
				http.Error(w,
					"invalid session token",
					http.StatusUnauthorized,
				)
				// Something weird happened.
				log.Printf("Failed to verify session "+
					"token after refresh: %#v", err)
				return
			case err != nil:
				http.Error(w, errors.Wrap(err,
					"failed to verify session "+
						"token after refresh",
				).Error(), http.StatusInternalServerError)
				log.Printf("Failed to verify session "+
					"token after refresh: %#v", err)
				return
			}

			// If we ran the gamut of possible errors here,
			// we're in the clear and the session was
			// refreshed.  Carry on.

			if len(ctrs) == 0 {
				// If there was no context, just apply
				// the handler.
				h(w, r, ps)
				return
			}

			// Otherwise, load the auth context and apply
			// the requested contexts to the request.
			ctx := new(auth.Context)
			err = db.View(auth.GetContext(ctx, token))
			switch {
			case auth.IsContextMissing(err):
				// Valid session with no context.
				log.Printf("unexpected error: valid "+
					"session %#q with no context",
					token,
				)
				http.Error(w, errors.Wrap(
					err, "error getting session context",
				).Error(), http.StatusInternalServerError)
				return
			case err != nil:
				http.Error(w,
					"error getting session context",
					http.StatusInternalServerError,
				)
				return
			}
			for _, ctr := range ctrs {
				r = ctr(r, ctx)
			}
			h(w, r, ps)
			return
		default:
			http.Error(w, errors.Wrap(
				err, "unexpected server error",
			).Error(), http.StatusInternalServerError)
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
		switch {
		case admin.IsNotFound(err):
			http.Error(w, err.Error(), http.StatusUnauthorized)
			return
		case err != nil:
			http.Error(w, errors.Wrap(err,
				"error authorizing admin API key",
			).Error(), http.StatusInternalServerError)
			return
		}

		h(w, r, ps)
	}
}

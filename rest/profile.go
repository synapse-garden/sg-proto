package rest

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/convo"
	mw "github.com/synapse-garden/sg-proto/rest/middleware"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream"
	"github.com/synapse-garden/sg-proto/stream/river"
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
func (p Profile) Bind(r *htr.Router) (Cleanup, error) {
	db := p.DB
	if db == nil {
		return nil, errors.New("nil Profile DB handle")
	}
	r.GET("/profile", mw.AuthUser(p.Get, db, mw.CtxSetUserID))
	r.DELETE("/profile", mw.AuthUser(p.Delete, db, mw.CtxSetUserID))

	return nil, nil
}

// Get fetches the user's Profile by userID.
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

// Delete deletes the User by ID from the UserBucket, disables the Login
// but retains it, and deletes all of the user's Sessions, Contexts, and
// Tokens.  It also hangs up all of the user's connected rivers.
func (p Profile) Delete(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	userID := mw.CtxGetUserID(r)

	// Prepare surveys to hang up each of the convos and streams.
	var survs []river.Surveyor
	// Which buckets belonged to which surveys?
	var cleanup [][]store.Bucket

	// Find all of the user's stateful rivers (convos and streams.)
	// Note that there are more hangups, but they don't store state
	// like streams and convos.
	err := p.View(func(tx *bolt.Tx) error {
		convos, err := convo.GetAll(userID)(tx)
		if err != nil {
			return err
		}

		strs, err := stream.GetAll(userID)(tx)
		if err != nil {
			return err
		}

		uBkt := store.Bucket(userID)

		// First find any hangups stored for the user's convos.
		for _, c := range convos {
			bkts := []store.Bucket{
				river.HangupBucket,
				store.Bucket(c.ID),
				uBkt,
			}
			surv, err := river.NewSurvey(tx,
				river.DefaultTimeout,
				bkts...,
			)

			switch {
			case river.IsStreamMissing(err):
				// It's not connected, no need to hang up.
			case err != nil:
				return err
			default:
				// Otherwise, add to the surveys.
				survs = append(survs, surv)
				cleanup = append(cleanup, bkts)
			}
		}

		// Next, find any hangups stored for the user's streams.
		for _, str := range strs {
			bkts := []store.Bucket{
				river.HangupBucket,
				store.Bucket(str.ID),
				uBkt,
			}
			surv, err := river.NewSurvey(tx,
				river.DefaultTimeout,
				bkts...,
			)

			switch {
			case river.IsStreamMissing(err):
				// It's not connected, no need to hang up.
			case err != nil:
				return err
			default:
				// Otherwise, add to the surveys.
				survs = append(survs, surv)
				cleanup = append(cleanup, bkts)
			}
		}

		// Finally, find any hangups for the user's notifs.
		bkts := []store.Bucket{
			river.HangupBucket,
			river.ResponderBucket,
			store.Bucket(userID),
		}
		surv, err := river.NewSurvey(tx,
			river.DefaultTimeout,
			bkts...,
		)
		switch {
		case river.IsStreamMissing(err):
			// Notifs not connected, no need to hang up.
		case err != nil:
			return err
		default:
			// Otherwise, add to the surveys.
			survs = append(survs, surv)
			cleanup = append(cleanup, bkts)
		}

		// Found everything OK.
		return nil
	})

	if err != nil {
		// There was a database error.  Fail.
		http.Error(w, errors.Wrap(
			err, "failed to prepare hangup surveys",
		).Error(), http.StatusInternalServerError)
		return
	}

	// Now, run all the queries concurrently and wait.
	errs := make(chan error, len(survs))
	for i, surv := range survs {
		go func(j int, s river.Surveyor, es chan<- error) {
			err := river.MakeSurvey(s, river.HUP, river.OK)
			if river.IsMissing(err) {
				err = survErr{e: err, bkts: cleanup[j]}
			}
			es <- err
		}(i, surv, errs)
	}

hangups:
	for i := 0; i < len(survs); i++ {
		var err error
		select {
		case err = <-errs:
		case <-time.After(river.MaxTimeout):
			msg := fmt.Sprintf("failed to make profile " +
				"DELETE hangup survey: " +
				"timed out",
			)
			log.Println(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		switch {
		case river.IsMissing(errors.Cause(err)):
			// Missing hangup.  Removed already?
			err = p.View(river.CheckMissing(
				err.(survErr).bkts...,
			))
			if err == nil {
				// Removed already.
				continue hangups
			}
			// Otherwise, failed.
			fallthrough

		case err != nil:
			msg := fmt.Sprintf("failed to make profile "+
				"DELETE hangup survey: "+
				"%s",
				err.Error())
			log.Println(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		// Hung up OK, check next.
	}

	err = p.Update(store.Wrap(
		users.Delete(userID),
		auth.Disable(userID),
	))

	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
}

package rest

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/synapse-garden/sg-proto/admin"
	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/convo"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/notif"
	mw "github.com/synapse-garden/sg-proto/rest/middleware"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream"
	"github.com/synapse-garden/sg-proto/stream/river"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	htr "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

var AdminNotifs = "admin"

// Admin implements API for an admin token and DB handle.
type Admin struct {
	auth.Token
	*bolt.DB
	river.Pub
}

// Bind implements API.Bind on Admin.
func (a *Admin) Bind(r *htr.Router) (Cleanup, error) {
	db := a.DB
	if db == nil {
		return nil, errors.New("nil Admin DB handle")
	}

	if a.Token != nil {
		// User wants to create a new token.
		err := db.Update(admin.NewToken(a.Token))
		if err != nil {
			return nil, err
		}
	} else if err := db.View(admin.CheckExists); err != nil {
		switch err.(type) {
		case admin.ErrNotFound:
			newToken := auth.Token(uuid.NewV4().Bytes())
			log.Printf("new admin key generated: %#q",
				base64.StdEncoding.EncodeToString(newToken))
			err = db.Update(admin.NewToken(newToken))
			if err != nil {
				return nil, err
			}
		default:
			return nil, errors.Wrap(err, "failed to check for existing admin key")
		}
	}

	err := db.Update(func(tx *bolt.Tx) (e error) {
		a.Pub, e = river.NewPub(AdminNotifs, NotifStream, tx)
		return
	})
	if err != nil {
		return nil, err
	}

	r.GET("/admin/verify", mw.AuthAdmin(a.Verify, db))
	r.GET("/admin/tickets", mw.AuthAdmin(a.GetTickets, db))
	r.POST("/admin/tickets", mw.AuthAdmin(a.NewTicket, db))
	r.GET("/admin/profiles", mw.AuthAdmin(a.GetAllProfiles, db))
	// PATCH /admin/profiles/bodie?addCoin=1000 (or -1000)
	r.PATCH("/admin/profiles/:id", mw.AuthAdmin(a.PatchProfile, db))
	// POST a new Login with corresponding User.
	r.POST("/admin/logins", mw.AuthAdmin(a.NewLogin, db))
	r.DELETE("/admin/tickets/:ticket", mw.AuthAdmin(a.DeleteTicket, db))
	r.DELETE("/admin/users/:user_id", mw.AuthAdmin(a.DeleteUser, db))

	return a.Cleanup, nil
}

// Cleanup closes the Admin's Pub river and deletes it from the DB.
func (a Admin) Cleanup() error {
	if err := a.Pub.Close(); err != nil {
		return err
	}
	return a.Update(func(tx *bolt.Tx) error {
		return river.DeletePub(AdminNotifs, NotifStream, tx)
	})
}

func (Admin) Verify(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	if err := json.NewEncoder(w).Encode(true); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
	}
}

func (a Admin) GetTickets(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	var (
		countStr = r.FormValue("count")
		count    = 20
		err      error
	)

	if len(countStr) != 0 {
		count, err = strconv.Atoi(countStr)
		switch {
		case err != nil:
			http.Error(w, errors.Wrapf(err, fmt.Sprintf(
				`invalid "count" value %#q`, countStr,
			)).Error(), http.StatusBadRequest)
			return
		case count < 1:
			http.Error(w, `invalid "count" value < 1`, http.StatusBadRequest)
			return
		}
	}

	var tkts []incept.Ticket
	err = a.View(func(tx *bolt.Tx) (e error) {
		tkts, e = incept.GetTickets(count)(tx)
		return
	})

	if err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to get tickets",
		).Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(tkts); err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to write response",
		).Error(), http.StatusInternalServerError)
		return
	}
}

func (a Admin) NewTicket(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	var (
		countStr = r.FormValue("count")
		count    = 1
		err      error
		db       = a.DB
	)
	if len(countStr) != 0 {
		count, err = strconv.Atoi(countStr)
		switch {
		case err != nil:
			http.Error(w, errors.Wrapf(err, fmt.Sprintf(
				`invalid "count" value %#q`, countStr,
			)).Error(), http.StatusBadRequest)
			return
		case count < 1:
			http.Error(w, `invalid "count" value < 1`, http.StatusBadRequest)
			return
		}
	}

	tkts := make([]incept.Ticket, count)
	result := make([]string, count)
	for i := range result {
		tkt := incept.Ticket(uuid.NewV4())
		tkts[i] = tkt
		result[i] = tkt.String()
	}

	if err := db.Update(incept.NewTickets(tkts...)); err != nil {
		result = nil
		http.Error(w, errors.Wrap(err, "failed to insert new tickets").Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		log.Printf("failed to marshal new tickets after writing to DB, trying to roll back: %#v", err)
		newErr := db.Update(incept.DeleteTickets(tkts...))
		if newErr != nil {
			log.Printf("failed to roll back "+
				"creation of new tickets "+
				"after error %#v: %#v",
				err, newErr)
			http.Error(w, errors.Wrapf(
				newErr, "failed to rollback "+
					"new tickets after "+
					"error: %s",
				err.Error()).Error(),
				http.StatusInternalServerError)
			return
		}
		result = nil
		http.Error(w, errors.Wrap(err, "failed to marshal new tickets after inserting").Error(), http.StatusInternalServerError)
		return
	}
}

func (a Admin) GetAllProfiles(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	all := users.Users{}
	if err := a.View(all.GetAll); err != nil {
		http.Error(w, errors.Wrap(err,
			"failed to get all profiles",
		).Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(all)
}

// PatchProfile is a PATCH handler for an Admin to add Coin to a given
// user's Profile.  The User is notified with the updated Profile value.
// The caller should use the URL parameter addCoin=<int64 coin amount>.
func (a Admin) PatchProfile(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	userID := ps.ByName("id")
	if err := a.View(users.CheckUsersExist(userID)); err != nil {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	}

	coinStr := r.FormValue("addCoin")
	if coinStr == "" {
		http.Error(w,
			"no value passed for addCoin parameter",
			http.StatusBadRequest,
		)
		return
	}

	coin, err := strconv.ParseInt(coinStr, 10, 64)
	if err != nil {
		http.Error(w, errors.Wrapf(err,
			"failed to parse coin value %s",
			coinStr).Error(), http.StatusBadRequest,
		)
		return
	}

	u := &users.User{Name: userID}
	err = a.Update(store.Wrap(
		users.CheckUsersExist(userID),
		users.AddCoin(u, coin),
	))
	switch {
	case users.IsMissing(err):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	notif.Encode(a.Pub, u, notif.MakeUserTopic(u.Name))
	json.NewEncoder(w).Encode(u)
}

// NewLogin allows an Admin to create a new Login without punching a
// Ticket.
func (a Admin) NewLogin(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	l := new(auth.Login)
	if err := json.NewDecoder(r.Body).Decode(l); err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to parse Login",
		).Error(), http.StatusBadRequest)
		return
	}

	err := incept.InceptNoTicket(l, a.DB)
	switch {
	case users.IsExists(err):
		http.Error(w, err.Error(), http.StatusConflict)
		return
	case auth.IsExists(err):
		http.Error(w, err.Error(), http.StatusConflict)
		return
	case err != nil:
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	json.NewEncoder(w).Encode(l.User)
}

func (a Admin) DeleteTicket(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	db := a.DB
	tStr := ps.ByName("ticket")
	ticket, err := uuid.FromString(tStr)
	if err != nil {
		http.Error(w, errors.Wrapf(err, fmt.Sprintf(
			"invalid ticket %#q", tStr)).Error(),
			http.StatusBadRequest)
		return
	}

	err = db.Update(incept.DeleteTickets(incept.Ticket(ticket)))
	if err != nil {
		http.Error(w, errors.Wrapf(err, fmt.Sprintf(
			"failed to delete ticket %#q", tStr,
		)).Error(), http.StatusInternalServerError)
		return
	}
}

func (a Admin) DeleteUser(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	userID := ps.ByName("user_id")

	err := a.View(users.CheckUsersExist(userID))
	switch {
	case users.IsMissing(err):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrap(
			err, "failed to query database",
		).Error(), http.StatusInternalServerError)
		return
	}

	// Prepare surveys to hang up each of the convos and streams.
	var survs []river.Surveyor
	// Which buckets belonged to which surveys?
	var cleanup [][]store.Bucket

	// Find all of the user's stateful rivers (convos and streams.)
	// Note that there are more hangups, but they don't store state
	// like streams and convos.
	// TODO:
	//   - Wrap this into a tighter store.Wrap compatible version
	//     with self-mutating collections
	//   - Put it someplace it can be reused
	err = a.View(func(tx *bolt.Tx) error {
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
			// TODO: Failed.
			msg := fmt.Sprintf("failed to make admin "+
				"DELETE hangup survey: "+
				"timed out\n for User %#q",
				userID)
			log.Println(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		switch {
		case river.IsMissing(errors.Cause(err)):
			// Missing hangup.  Removed already?
			err = a.View(river.CheckMissing(
				err.(survErr).bkts...,
			))
			if err == nil {
				// Removed already.
				continue hangups
			}
			// Otherwise, failed.
			fallthrough

		case err != nil:
			// Some other problem.
			msg := fmt.Sprintf("failed to make admin "+
				"DELETE hangup survey: "+
				"%s\n for User %#q",
				err.Error(), userID)
			log.Println(msg)
			http.Error(w, msg, http.StatusInternalServerError)
			return
		}

		// Hung up OK, check next.
	}

	err = a.Update(store.Wrap(
		users.Delete(userID),
		auth.Disable(userID),
	))

	if err != nil {
		http.Error(w, errors.Wrap(err,
			"failed to disable user's login",
		).Error(), http.StatusInternalServerError)
		return
	}
}

type survErr struct {
	e    error
	bkts []store.Bucket
}

func (s survErr) Error() string { return s.e.Error() }
func (s survErr) Cause() error  { return s.e }

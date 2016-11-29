package rest

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"

	"github.com/synapse-garden/sg-proto/admin"
	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/incept"
	mw "github.com/synapse-garden/sg-proto/rest/middleware"

	"github.com/boltdb/bolt"
	htr "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// Admin implements API for an admin token and DB handle.
type Admin struct {
	auth.Token
	*bolt.DB
}

// Bind implements API.Bind on Admin.
func (a Admin) Bind(r *htr.Router) error {
	db := a.DB
	if db == nil {
		return errors.New("Admin DB handle must not be nil")
	}

	if a.Token != nil {
		// User wants to create a new token.
		err := db.Update(admin.NewToken(a.Token))
		if err != nil {
			return err
		}
	} else if err := db.View(admin.CheckExists); err != nil {
		switch err.(type) {
		case admin.ErrNotFound:
			newToken := auth.Token(uuid.NewV4().Bytes())
			log.Printf("new admin key generated: %#q",
				base64.StdEncoding.EncodeToString(newToken))
			err = db.Update(admin.NewToken(newToken))
			if err != nil {
				return err
			}
		default:
			return errors.Wrap(err, "failed to check for existing admin key")
		}
	}

	r.GET("/admin/verify", mw.AuthAdmin(a.Verify, db))
	r.POST("/admin/tickets", mw.AuthAdmin(a.NewTicket, db))
	r.DELETE("/admin/tickets/:ticket", mw.AuthAdmin(a.DeleteTicket, db))

	return nil
}

func (Admin) Verify(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	if err := json.NewEncoder(w).Encode(true); err != nil {
		http.Error(w, "failed to write response", http.StatusInternalServerError)
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

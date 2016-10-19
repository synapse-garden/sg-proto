package rest

import (
	"log"
	"net/http"

	"github.com/synapse-garden/sg-proto/notif"
	mw "github.com/synapse-garden/sg-proto/rest/middleware"
	"github.com/synapse-garden/sg-proto/rest/ws"
	"github.com/synapse-garden/sg-proto/stream/river"

	"github.com/boltdb/bolt"
	htr "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	xws "golang.org/x/net/websocket"
)

// Notif is a websocket API endpoint for authenticated users.
func Notif(r *htr.Router, db *bolt.DB) error {
	// When a client wants to connect to notifs, use stream.NewSub.
	r.GET("/notifs", mw.AuthUser(ConnectNotifs(db), db, mw.CtxSetUserID))
	return nil
}

// ConnectNotifs binds a subscriber River and serves it over a Websocket.
func ConnectNotifs(db *bolt.DB) htr.Handle {
	return func(w http.ResponseWriter, r *http.Request, _ htr.Params) {
		userID := mw.CtxGetUserID(r)
		var read river.Sub
		err := db.Update(func(tx *bolt.Tx) (e error) {
			read, e = river.NewSub(
				notif.River,
				tx,
				notif.Topics(userID)...,
			)
			return
		})

		switch {
		case river.IsStreamMissing(err):
			http.Error(w, errors.Wrap(err,
				"subscription server not found",
			).Error(), http.StatusNotFound)
			return
		case err != nil:
			log.Printf("ERROR: unexpected river error: %s", err.Error())
			http.Error(w, errors.Wrap(err,
				"unexpected river error",
			).Error(), http.StatusInternalServerError)
			return
		}

		xws.Server{
			Handshake: ws.Check,
			Handler:   ws.BindRead(read),
		}.ServeHTTP(w, r)

		if err := read.Close(); err != nil {
			log.Printf("ERROR: failed to Close River: %s", err.Error())
			http.Error(w, "failed to close River", http.StatusInternalServerError)
		}
	}
}

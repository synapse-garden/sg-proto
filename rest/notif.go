package rest

import (
	"log"
	"net/http"

	"github.com/synapse-garden/sg-proto/notif"
	mw "github.com/synapse-garden/sg-proto/rest/middleware"
	"github.com/synapse-garden/sg-proto/rest/ws"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream/river"

	"github.com/boltdb/bolt"
	htr "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	xws "golang.org/x/net/websocket"
)

// NotifStream is the streamID to be used for notifs.
const NotifStream = "notifs"

// Notif implements API.  It handles websocket connections to API
// endpoint notification publishers.  A user will receive events on the
// websocket when an API publishes a notification event to its Pub.
type Notif struct {
	*bolt.DB
}

// Bind implements API.Bind on Notif.
func (n Notif) Bind(r *htr.Router) error {
	if n.DB == nil {
		return errors.New("Notif DB handle must not be nil")
	}
	// When a client wants to connect to notifs, use stream.NewSub.
	r.GET("/notifs", mw.AuthWSUser(n.Connect, n.DB, mw.CtxSetUserID))
	return nil
}

// Connect binds a subscriber River and serves it over a Websocket.
func (n Notif) Connect(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	userID := mw.CtxGetUserID(r)

	// Create a new river.Responder to respond to hangup
	// requests from the backend.
	var rsp river.Responder
	err := n.Update(func(tx *bolt.Tx) (e error) {
		rsp, e = river.NewResponder(tx,
			river.HangupBucket,
			river.ResponderBucket,
			store.Bucket(userID),
		)
		return
	})
	if err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to start new River",
		).Error(), http.StatusInternalServerError)
		return
	}

	var read river.Sub
	err = n.Update(func(tx *bolt.Tx) (e error) {
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

	h := ws.MakeHangupRecver(rsp, read)
	errCh := make(chan error)
	go func() {
		// Start a survey waiting for hangup.
		errCh <- river.AwaitHangup(h)
	}()

	xws.Server{
		Handshake: ws.Check,
		Handler:   ws.BindRead(h.Recver()),
	}.ServeHTTP(w, r)

	err = n.Update(func(tx *bolt.Tx) error {
		read.Close()
		rsp.Close()
		<-errCh

		return river.DeleteResp(tx, h.ID(),
			river.HangupBucket,
			river.ResponderBucket,
			store.Bucket(userID),
		)
	})
	if err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to clean up notif River",
		).Error(), http.StatusInternalServerError)
		return
	}
}

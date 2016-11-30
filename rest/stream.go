package rest

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	mw "github.com/synapse-garden/sg-proto/rest/middleware"
	"github.com/synapse-garden/sg-proto/rest/ws"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream"
	"github.com/synapse-garden/sg-proto/stream/river"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	htr "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	xws "golang.org/x/net/websocket"
)

// Stream sets up the Streams API on the Router for the given DB.
type Stream struct {
	*bolt.DB
}

// Bind implements API.Bind on Stream.
func (s Stream) Bind(r *htr.Router) error {
	db := s.DB
	if db == nil {
		return errors.New("Stream DB handle must not be nil")
	}

	if err := db.Update(river.ClearRivers); err != nil {
		return err
	}
	// vx.y.0:
	//   - TODO: assign work, pin version
	//   - Fractal / graph Stream
	//   - Can compose Streams
	//   - A Stream is a sort of concurrent B-tree where updates
	//     can happen asynchronously as long as they don't overlap
	// v0.1.0:
	//   - Topics
	//   - Mangos inproc?
	//   - POST a  new Stream
	//   - PUT a stream by ID (to add Writers for example)
	//   - GET a stream by ID
	//     - What comes back?  A JSON object, or a WS conn?
	//     - First message: big grump lump of all relevant (timely)
	//       messages
	//     - Next messages: incoming messages / updates
	//   - GET all my streams
	//     - Streams I have created
	//     - Streams other people have added me to
	//   - DELETE a stream I own
	//   - POST new stream topic /streams/:topic/:user_id?
	// v0.0.1:
	//   - POST to user_id to create a new conversation, can only
	//     have one convo between any two users
	//   - POST to existing convo just returns the existing one
	//   - The REST endpoint receives a websocket connection that
	//     first sends an update with everything "current"
	//   - GET does the same thing but doesn't try to create a new
	//     convo
	//   - GET on /streams/:user_id GETs any streams from that user
	//   - GET on /streams returns metadata of all convos I own
	//   - Just slices of {username, timestamp, message}
	//   - DELETE a stream I own

	r.GET("/streams/:stream_id/start", mw.AuthWSUser(
		s.Connect,
		db, mw.CtxSetUserID,
	))

	r.POST("/streams", mw.AuthUser(
		s.Create,
		db, mw.CtxSetUserID,
	))

	r.GET("/streams", mw.AuthUser(
		s.GetAll,
		db, mw.CtxSetUserID,
	))

	r.GET("/streams/:stream_id", mw.AuthUser(
		s.Get,
		db, mw.CtxSetUserID,
	))

	r.PUT("/streams/:stream_id", mw.AuthUser(
		s.Put,
		db, mw.CtxSetUserID,
	))

	r.DELETE("/streams/:stream_id", mw.AuthUser(
		s.Delete,
		db, mw.CtxSetUserID,
	))

	return nil
}

// Connect is a Handle which opens and binds a WebSocket session to a
// Stream.  The messages are transported as-is.
func (s Stream) Connect(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	userID := mw.CtxGetUserID(r)
	id := ps.ByName("stream_id")
	if _, err := uuid.FromString(id); err != nil {
		http.Error(w, errors.Wrapf(
			err, "invalid stream ID %#q", id,
		).Error(), http.StatusBadRequest)
		return
	}

	str := new(stream.Stream)
	err := s.View(stream.Get(str, id))
	switch {
	case stream.IsMissing(err):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrapf(
			err, "failed to get stream %#q", id,
		).Error(), http.StatusInternalServerError)
		return
	case !str.Writers[userID]:
		// TODO: Add read-only Streams
		http.Error(w, fmt.Sprintf(
			"user %#q cannot write to stream %#q",
			userID, id,
		), http.StatusUnauthorized)
		return
	}

	// Create a new river.Responder to respond to hangup requests
	// from the backend.
	var rsp river.Responder
	err = s.Update(func(tx *bolt.Tx) (e error) {
		rsp, e = river.NewResponder(tx,
			river.HangupBucket,
			store.Bucket(str.ID),
			store.Bucket(userID),
		)
		return
	})
	switch {
	case river.IsExists(err):
		http.Error(w, errors.Wrap(
			err, "failed to start new River",
		).Error(), http.StatusConflict)
		return
	case err != nil:
		http.Error(w, errors.Wrap(
			err, "failed to start new River",
		).Error(), http.StatusInternalServerError)
		return
	}

	h := ws.MakeHangup(rsp, nil)
	errCh := make(chan error)
	go func() {
		// Start a survey waiting for hangup.
		errCh <- river.AwaitHangup(h)
	}()

	var rv river.Bus
	err = s.Update(func(tx *bolt.Tx) (e error) {
		rv, e = river.NewBus(userID, str.ID, tx)
		return
	})

	switch {
	case river.IsExists(err):
		http.Error(w, errors.Wrap(
			err, "failed to start new River",
		).Error(), http.StatusConflict)
		return
	case err != nil:
		http.Error(w, errors.Wrap(
			err, "failed to start new River",
		).Error(), http.StatusInternalServerError)
		return
	}

	xws.Server{
		Handshake: ws.Check,
		// Use the HangupSender.Read to hang up the river if a
		// hangup survey is received.
		Handler: ws.Bind(rv, h.Read),
	}.ServeHTTP(w, r)

	err = s.Update(func(tx *bolt.Tx) (e error) {
		eD := river.DeleteBus(userID, str.ID, rv.ID())(tx)
		eC := rv.Close()
		switch {
		case eD != nil && eC != nil:
			e = errors.Wrap(eC, eD.Error())
			return
		case eD != nil:
			e = eD
			return
		case eC != nil:
			e = eC
			return
		}
		eD = river.DeleteResp(tx, h.ID(),
			river.HangupBucket,
			store.Bucket(str.ID),
			store.Bucket(userID),
		)
		eC = rsp.Close()
		<-errCh
		switch {
		case eD != nil && eC != nil:
			e = errors.Wrap(eC, eD.Error())
		case eD != nil:
			e = eD
		case eC != nil:
			e = eC
		}
		return
	})
	if err != nil {
		http.Error(w, "failed to clean up River",
			http.StatusInternalServerError)
		log.Fatalf("ERROR: %s", errors.Wrapf(
			err, "failed to clean up River %#q", id,
		).Error())
	}

	// Notify stream members that the user has left.
	for u := range str.Readers {
		err = notif.Encode(s.Pub, stream.Disconnected(userID), notif.MakeUserTopic(u))
		if err != nil {
			log.Printf("failed to notify user %q of stream leave", u)
		}
	}
}

// Create is a Handle over the DB which checks that the POSTed Stream is
// valid and then creates it, returning the created Stream.
func (s Stream) Create(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	str := new(stream.Stream)
	if err := json.NewDecoder(r.Body).Decode(str); err != nil {
		http.Error(w, errors.Wrap(
			err, "malformed Stream",
		).Error(), http.StatusBadRequest)
		return
	}

	userID := mw.CtxGetUserID(r)
	id := uuid.NewV4().String()

	str.Owner = userID
	str.ID = id

	allUsers := make([]string, len(str.Readers)+len(str.Writers)+1)
	allUsers[0] = userID
	next := 1
	for r := range str.Readers {
		allUsers[next] = r
		next++
	}
	for w := range str.Writers {
		allUsers[next] = w
		next++
	}

	err := s.View(store.Wrap(
		stream.CheckNotExist(id),
		users.CheckUsersExist(allUsers...),
	))
	if err != nil {
		msg := errors.Wrap(err, "failed to check Stream").Error()
		var code int
		switch {
		case users.IsMissing(err):
			code = http.StatusNotFound
		case stream.IsExists(err):
			code = http.StatusConflict
		default:
			code = http.StatusInternalServerError
		}
		http.Error(w, msg, code)
		return
	}

	err = s.Update(store.Wrap(
		stream.CheckNotExist(id),
		users.CheckUsersExist(allUsers...),
		stream.Upsert(str),
	))
	if err != nil {
		msg := errors.Wrap(err, "failed to create Stream").Error()
		var code int
		switch {
		case stream.IsExists(err):
			code = http.StatusConflict
		case users.IsMissing(err):
			code = http.StatusNotFound
		default:
			code = http.StatusInternalServerError
		}
		http.Error(w, msg, code)
		return
	}

	if err := json.NewEncoder(w).Encode(str); err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to write Stream to user",
		).Error(), http.StatusInternalServerError)
		return
	}
}

// Put is a Handle which updates a Stream in the DB by ID.
func (s Stream) Put(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	userID := mw.CtxGetUserID(r)
	id := ps.ByName("stream_id")
	if _, err := uuid.FromString(id); err != nil {
		http.Error(w, errors.Wrapf(
			err, "invalid stream ID %#q", id,
		).Error(), http.StatusBadRequest)
		return
	}
	str := new(stream.Stream)
	if err := json.NewDecoder(r.Body).Decode(str); err != nil {
		http.Error(w, errors.Wrap(
			err, "malformed Stream",
		).Error(), http.StatusBadRequest)
		return
	}

	str.Owner = userID
	str.ID = id

	allUsers := make([]string, len(str.Readers)+len(str.Writers)+1)
	allUsers[0] = userID
	next := 1
	for r := range str.Readers {
		allUsers[next] = r
		next++
	}
	for w := range str.Writers {
		allUsers[next] = w
		next++
	}

	err := s.View(store.Wrap(
		stream.CheckExists(id),
		users.CheckUsersExist(allUsers...),
		// TODO: FIXME: Make sure user is authorized.
	))
	if err != nil {
		msg := errors.Wrap(err, "failed to check new Stream").Error()
		var code int
		switch {
		case users.IsMissing(err), stream.IsMissing(err):
			code = http.StatusNotFound
		default:
			code = http.StatusInternalServerError
		}
		http.Error(w, msg, code)
		return
	}

	existing := new(stream.Stream)
	err = s.View(stream.Get(existing, id))
	switch {
	case stream.IsMissing(err):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrapf(
			err, "failed to get stream %#q", id,
		).Error(), http.StatusInternalServerError)
		return
	}

	// TODO: FIXME: Hang up removed read / write users
	err = s.Update(store.Wrap(
		stream.CheckExists(id),
		users.CheckUsersExist(allUsers...),
		stream.Upsert(str),
	))
	if err != nil {
		msg := errors.Wrap(
			err, "failed to upsert Stream",
		).Error()
		var code int
		switch {
		case stream.IsMissing(err), users.IsMissing(err):
			code = http.StatusNotFound
		default:
			code = http.StatusInternalServerError
		}
		http.Error(w, msg, code)
		return
	}

	if err := json.NewEncoder(w).Encode(str); err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to write Stream to user",
		).Error(), http.StatusInternalServerError)
		return
	}
}

// GetAll is a Handle which writes all Streams owned by the user to the
// ResponseWriter.
//
// TODO: Make Filters more flexible so users who aren't Owners can also
//       get Streams they belong to.
func (s Stream) GetAll(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	// TODO: add search parameters
	// TODO: add pagination
	userID := mw.CtxGetUserID(r)
	var allStreams []*stream.Stream
	err := s.View(func(tx *bolt.Tx) (e error) {
		allStreams, e = stream.GetAll(userID)(tx)
		return
	})
	if err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to get Streams",
		).Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(allStreams); err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to encode Streams").Error(),
			http.StatusInternalServerError,
		)
		return
	}
}

// Get is a Handle which gets the given Stream by ID.  Any user who is
// an Owner, Reader, or Writer can get a Stream by ID.
func (s Stream) Get(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	userID := mw.CtxGetUserID(r)
	id := ps.ByName("stream_id")
	if _, err := uuid.FromString(id); err != nil {
		http.Error(w, errors.Wrapf(
			err, "invalid stream ID %#q", id,
		).Error(), http.StatusBadRequest)
		return
	}
	existing := new(stream.Stream)
	err := s.View(stream.Get(existing, id))
	switch {
	case stream.IsMissing(err):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrapf(
			err, "failed to get stream %#q", id,
		).Error(), http.StatusInternalServerError)
		return
	case existing.Owner != userID &&
		!existing.Readers[userID] &&
		!existing.Writers[userID]:
		http.Error(w, fmt.Sprintf(
			"user %#q not a member of stream %#q",
			id, userID,
		), http.StatusUnauthorized)
		return
	}

	if err := json.NewEncoder(w).Encode(existing); err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to encode Stream").Error(),
			http.StatusInternalServerError,
		)
		return
	}
}

// Delete deletes the given Stream by ID.
func (s Stream) Delete(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	userID := mw.CtxGetUserID(r)
	id := ps.ByName("stream_id")
	if _, err := uuid.FromString(id); err != nil {
		http.Error(w, errors.Wrapf(
			err, "invalid stream ID %#q", id,
		).Error(), http.StatusBadRequest)
		return
	}
	existing := new(stream.Stream)
	err := s.View(stream.Get(existing, id))
	switch {
	case stream.IsMissing(err):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrapf(
			err, "failed to get stream %#q", id,
		).Error(), http.StatusInternalServerError)
		return
	case existing.Owner != userID:
		http.Error(w, fmt.Sprintf(
			"stream %#q not owned by user %#q",
			id, userID,
		), http.StatusUnauthorized)
		return
	}

	return
}

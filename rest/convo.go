package rest

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"

	"github.com/synapse-garden/sg-proto/convo"
	mw "github.com/synapse-garden/sg-proto/rest/middleware"
	"github.com/synapse-garden/sg-proto/rest/ws"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream/river"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	htr "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	xws "golang.org/x/net/websocket"
)

// Convo implements API.  It manages Convos.
type Convo struct {
	*bolt.DB
}

// Bind implements API.Bind on Convo.
func (c Convo) Bind(r *htr.Router) error {
	db := c.DB
	if db == nil {
		return errors.New("Convo DB handle must not be nil")
	}

	if err := db.Update(river.ClearRivers); err != nil {
		return err
	}

	r.GET("/convos/:convo_id/start", mw.AuthWSUser(
		c.Connect,
		db, mw.CtxSetUserID,
	))

	r.GET("/convos/:convo_id/messages", mw.AuthUser(
		c.GetMessages,
		db, mw.CtxSetUserID,
	))

	r.POST("/convos", mw.AuthUser(
		c.Create,
		db, mw.CtxSetUserID,
	))

	r.GET("/convos", mw.AuthUser(
		c.GetAll,
		db, mw.CtxSetUserID,
	))

	r.GET("/convos/:convo_id", mw.AuthUser(
		c.Get,
		db, mw.CtxSetUserID,
	))

	r.PUT("/convos/:convo_id", mw.AuthUser(
		c.Update,
		db, mw.CtxSetUserID,
	))

	r.DELETE("/convos/:convo_id", mw.AuthUser(
		c.Delete,
		db, mw.CtxSetUserID,
	))

	return nil
}

// Connect returns a Handle which opens and binds a WebSocket session to
// a Convo.  The messages are transported as-is.
func (c Convo) Connect(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	db := c.DB

	userID := mw.CtxGetUserID(r)
	id := ps.ByName("convo_id")
	if _, err := uuid.FromString(id); err != nil {
		http.Error(w, errors.Wrapf(
			err, "invalid convo ID %#q", id,
		).Error(), http.StatusBadRequest)
		return
	}

	conv := new(convo.Convo)
	err := db.View(convo.Get(conv, id))
	switch {
	case convo.IsMissing(err):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrapf(
			err, "failed to get convo %#q", id,
		).Error(), http.StatusInternalServerError)
		return
	case !conv.Writers[userID]:
		http.Error(w, fmt.Sprintf(
			"user %#q cannot write to convo %#q",
			userID, id,
		), http.StatusUnauthorized)
		return
	}

	// Create a new river.Responder to respond to hangup
	// requests from the backend.
	var rsp river.Responder
	err = db.Update(func(tx *bolt.Tx) (e error) {
		rsp, e = river.NewResponder(tx,
			river.HangupBucket,
			store.Bucket(conv.ID),
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

	h := ws.MakeHangup(rsp, convo.Sender(userID).Read)
	errCh := make(chan error)
	go func() {
		// Start a survey waiting for hangup.
		errCh <- river.AwaitHangup(h)
	}()

	// If no Scribe, create one for the convo.
	var (
		scr   = convo.Scribe(conv.ID)
		scrID uint64
		first bool
	)
	err = db.Update(func(tx *bolt.Tx) (e error) {
		scrID, first, e = scr.Checkin(tx)
		return
	})
	if err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to check into convo",
		).Error(), http.StatusInternalServerError)
		return
	}

	if first {
		if err = db.Update(scr.Spawn); err != nil {
			http.Error(w, errors.Wrap(
				err, "failed to spawn Scribe",
			).Error(), http.StatusInternalServerError)
			return
		}
	}

	// Create a Bus to connect to the convo.
	var rv river.Bus
	err = db.Update(func(tx *bolt.Tx) (e error) {
		rv, e = river.NewBus(userID, conv.ID, tx)
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
		// Use the HangupSender.Read to hang up the
		// river if a hangup survey is received.
		Handler: ws.Bind(rv, h.Read),
	}.ServeHTTP(w, r)

	var last bool
	err = db.Update(func(tx *bolt.Tx) (e error) {
		last, e = scr.Checkout(scrID, tx)
		return
	})
	if err != nil {
		log.Fatal(errors.Wrap(err,
			"failed to check out of convo",
		).Error())
	}
	if last {
		if err := scr.Hangup(db); err != nil {
			log.Fatal(errors.Wrap(err,
				"failed to hangup Scribe",
			).Error())
		}
	}

	err = db.Update(func(tx *bolt.Tx) (e error) {
		eD := river.DeleteBus(userID, conv.ID, rv.ID())(tx)
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
			store.Bucket(conv.ID),
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
		http.Error(w,
			"failed to clean up River",
			http.StatusInternalServerError)
		log.Fatalf("ERROR: %s", errors.Wrapf(
			err, "failed to clean up River %#q", id,
		).Error())
	}
}

// GetMessages gets an array of convo.Message.  It can have filters
// applied in order to get a specific range of Messages.
func (c Convo) GetMessages(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	db := c.DB

	convoID := ps.ByName("convo_id")

	var result []convo.Message
	userID := mw.CtxGetUserID(r)

	conv := new(convo.Convo)
	err := db.View(convo.Get(conv, convoID))
	switch {
	case convo.IsMissing(err):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrapf(
			err, "failed to get convo %#q", convoID,
		).Error(), http.StatusInternalServerError)
		return
	case !conv.Readers[userID]:
		http.Error(w, fmt.Sprintf(
			"user %#q cannot write to convo %#q",
			userID, convoID,
		), http.StatusUnauthorized)
		return
	}

	err = db.View(func(tx *bolt.Tx) (e error) {
		result, e = convo.GetMessages(convoID, tx)
		return
	})
	switch {
	case convo.IsMissing(err):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrapf(
			err, "failed to get convo %#q", convoID,
		).Error(), http.StatusInternalServerError)
		return
	}

	if err := json.NewEncoder(w).Encode(result); err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to write Convo to user",
		).Error(), http.StatusInternalServerError)
		return
	}
}

// Create returns a Handle over the DB which checks that the POSTed
// Convo is valid and then creates it, returning the created Convo.
func (c Convo) Create(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	db := c.DB

	str := new(convo.Convo)
	if err := json.NewDecoder(r.Body).Decode(str); err != nil {
		http.Error(w, errors.Wrap(
			err, "malformed Convo",
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

	err := db.View(store.Wrap(
		convo.CheckNotExist(id),
		users.CheckUsersExist(allUsers...),
	))
	if err != nil {
		msg := errors.Wrap(
			err, "failed to check Convo",
		).Error()
		var code int
		switch {
		case users.IsMissing(err):
			code = http.StatusNotFound
		case convo.IsExists(err):
			code = http.StatusConflict
		default:
			code = http.StatusInternalServerError
		}
		http.Error(w, msg, code)
		return
	}

	err = db.Update(store.Wrap(
		convo.CheckNotExist(id),
		users.CheckUsersExist(allUsers...),
		convo.Upsert(str),
	))
	if err != nil {
		msg := errors.Wrap(
			err, "failed to create Convo",
		).Error()
		var code int
		switch {
		case convo.IsExists(err):
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
			err, "failed to write Convo to user",
		).Error(), http.StatusInternalServerError)
		return
	}
}

// Update returns a Handle which updates a Convo in the DB by ID.
func (c Convo) Update(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	db := c.DB

	userID := mw.CtxGetUserID(r)
	id := ps.ByName("convo_id")
	if _, err := uuid.FromString(id); err != nil {
		http.Error(w, errors.Wrapf(
			err, "invalid convo ID %#q", id,
		).Error(), http.StatusBadRequest)
		return
	}
	str := new(convo.Convo)
	if err := json.NewDecoder(r.Body).Decode(str); err != nil {
		http.Error(w, errors.Wrap(
			err, "malformed Convo",
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

	err := db.View(store.Wrap(
		convo.CheckExists(id),
		users.CheckUsersExist(allUsers...),
	))
	if err != nil {
		msg := errors.Wrap(
			err, "failed to check new Convo",
		).Error()
		var code int
		switch {
		case users.IsMissing(err), convo.IsMissing(err):
			code = http.StatusNotFound
		default:
			code = http.StatusInternalServerError
		}
		http.Error(w, msg, code)
		return
	}

	existing := new(convo.Convo)
	if err := db.View(convo.Get(existing, id)); convo.IsMissing(err) {
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	} else if err != nil {
		http.Error(w, errors.Wrapf(
			err, "failed to get convo %#q", id,
		).Error(), http.StatusInternalServerError)
		return
	}

	// TODO: FIXME: Hang up removed read / write users
	err = db.Update(store.Wrap(
		convo.CheckExists(id),
		users.CheckUsersExist(allUsers...),
		convo.Upsert(str),
	))
	if err != nil {
		msg := errors.Wrap(
			err, "failed to upsert Convo",
		).Error()
		var code int
		switch {
		case convo.IsMissing(err), users.IsMissing(err):
			code = http.StatusNotFound
		default:
			code = http.StatusInternalServerError
		}
		http.Error(w, msg, code)
		return
	}

	if err := json.NewEncoder(w).Encode(str); err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to write Convo to user",
		).Error(), http.StatusInternalServerError)
		return
	}
}

// GetAll returns a Handle which writes all Convos owned by the user to
// the ResponseWriter.
//
// TODO: Make Filters more flexible so users who aren't Owners can also
//       get Convos they belong to.
func (c Convo) GetAll(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	db := c.DB

	// TODO: add search parameters
	// TODO: add pagination
	userID := mw.CtxGetUserID(r)
	var allConvos []*convo.Convo
	err := db.View(func(tx *bolt.Tx) (e error) {
		allConvos, e = convo.GetAll(userID)(tx)
		return
	})
	if err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to get Convos",
		).Error(), http.StatusInternalServerError)
		return
	}
	if err := json.NewEncoder(w).Encode(allConvos); err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to encode Convos").Error(),
			http.StatusInternalServerError,
		)
		return
	}
}

// Get returns a Handle which gets the given Convo by ID.  Any user who
// is an Owner, Reader, or Writer can get a Convo by ID.
func (c Convo) Get(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	db := c.DB

	userID := mw.CtxGetUserID(r)
	id := ps.ByName("convo_id")
	if _, err := uuid.FromString(id); err != nil {
		http.Error(w, errors.Wrapf(
			err, "invalid convo ID %#q", id,
		).Error(), http.StatusBadRequest)
		return
	}
	existing := new(convo.Convo)
	err := db.View(convo.Get(existing, id))
	switch {
	case convo.IsMissing(err):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrapf(
			err, "failed to get convo %#q", id,
		).Error(), http.StatusInternalServerError)
		return
	case existing.Owner != userID &&
		!existing.Readers[userID] &&
		!existing.Writers[userID]:
		http.Error(w, fmt.Sprintf(
			"user %#q not a member of convo %#q",
			id, userID,
		), http.StatusUnauthorized)
		return
	}

	if err := json.NewEncoder(w).Encode(existing); err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to encode Convo").Error(),
			http.StatusInternalServerError,
		)
		return
	}
}

// Delete deletes the given Convo by ID.
func (c Convo) Delete(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	db := c.DB

	userID := mw.CtxGetUserID(r)
	id := ps.ByName("convo_id")
	if _, err := uuid.FromString(id); err != nil {
		http.Error(w, errors.Wrapf(
			err, "invalid convo ID %#q", id,
		).Error(), http.StatusBadRequest)
		return
	}
	existing := new(convo.Convo)
	err := db.View(convo.Get(existing, id))
	switch {
	case convo.IsMissing(err):
		http.Error(w, err.Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrapf(
			err, "failed to get convo %#q", id,
		).Error(), http.StatusInternalServerError)
		return
	case existing.Owner != userID:
		http.Error(w, fmt.Sprintf(
			"convo %#q not owned by user %#q",
			id, userID,
		), http.StatusUnauthorized)
		return
	}

	if err := c.Update(convo.Delete([]byte(id))); err != nil {
		http.Error(w, fmt.Sprintf(
			"failed to delete convo %#q: %s",
			id, err.Error(),
		), http.StatusInternalServerError)
		return
	}
}

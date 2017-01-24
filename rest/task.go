package rest

import (
	"encoding/json"
	"net/http"
	"net/url"
	"sort"
	"time"

	"github.com/synapse-garden/sg-proto/notif"
	mw "github.com/synapse-garden/sg-proto/rest/middleware"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream/river"
	"github.com/synapse-garden/sg-proto/task"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	htr "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const TaskNotifs = "tasks"

type Task struct {
	*bolt.DB
	river.Pub
}

func (t *Task) Bind(r *htr.Router) error {
	if t.DB == nil {
		return errors.New("Bind called with nil DB handle")
	}

	err := t.Update(func(tx *bolt.Tx) (e error) {
		t.Pub, e = river.NewPub(TaskNotifs, NotifStream, tx)
		return
	})
	if err != nil {
		return err
	}

	r.GET("/tasks", mw.AuthUser(
		t.GetAll,
		t.DB,
		mw.CtxSetUserID,
	))

	r.POST("/tasks", mw.AuthUser(
		t.Create,
		t.DB,
		mw.CtxSetUserID,
	))

	r.GET("/tasks/:id", mw.AuthUser(
		t.Get,
		t.DB,
		mw.CtxSetUserID,
	))

	r.DELETE("/tasks/:id", mw.AuthUser(
		t.Delete,
		t.DB,
		mw.CtxSetUserID,
	))

	r.PUT("/tasks/:id", mw.AuthUser(
		t.Put,
		t.DB,
		mw.CtxSetUserID,
	))

	return nil
}

func (t *Task) GetAll(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	vals, err := url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		http.Error(w, errors.Wrap(
			err, "bad query parameters",
		).Error(), http.StatusBadRequest)
		return
	}

	now := time.Now()

	filters := []task.Filter{}
	for k, v := range vals {
		switch {
		case k == "overdue" && len(v) > 0:
			switch v[0] {
			case "true":
				filters = append(filters, task.Overdue(now))
			case "false":
				filters = append(filters, task.NotYetDue(now))
			default:
				http.Error(w, errors.Errorf(
					`bad value %q for query `+
						`parameter %q`, v[0], k,
				).Error(), http.StatusBadRequest)
				return
			}
		case k == "complete" && len(v) > 0:
			switch v[0] {
			case "true":
				filters = append(filters, task.Complete)
			case "false":
				filters = append(filters, task.Incomplete)
			default:
				http.Error(w, errors.Errorf(
					`bad value %q for query `+
						`parameter %q`, v[0], k,
				).Error(), http.StatusBadRequest)
				return
			}
		default:
			http.Error(w, errors.Errorf(
				"unknown query parameter %q", k,
			).Error(), http.StatusBadRequest)
			return
		}
	}

	var ts []*task.Task
	err = t.View(func(tx *bolt.Tx) (e error) {
		ts, e = task.GetAll(
			mw.CtxGetUserID(r),
			filters...,
		)(tx)
		return
	})

	if err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to get tasks",
		).Error(), http.StatusInternalServerError)
		return
	}

	sort.Sort(task.ByOldest(ts))

	json.NewEncoder(w).Encode(ts)
}

func (t *Task) Create(w http.ResponseWriter, r *http.Request, _ htr.Params) {
	userID := mw.CtxGetUserID(r)

	tsk := new(task.Task)

	if err := json.NewDecoder(r.Body).Decode(tsk); err != nil {
		http.Error(w, errors.Wrap(
			err, "bad body",
		).Error(), http.StatusBadRequest)
		return
	}

	if tsk.Owner != userID {
		http.Error(w, errors.Errorf(
			"user %#q cannot create task for user %#q",
			userID, tsk.Owner,
		).Error(), http.StatusUnauthorized)
		return
	}

	// Make sure the Owner is in the readers / writers.
	tsk.Readers[userID] = true
	tsk.Writers[userID] = true

	allUsers := make([]string, len(tsk.Readers)+len(tsk.Writers)+1)
	allUsers[0] = userID
	next := 1
	for r := range tsk.Readers {
		allUsers[next] = r
		next++
	}
	for w := range tsk.Writers {
		allUsers[next] = w
		next++
	}

	err := t.View(users.CheckUsersExist(allUsers...))
	if err != nil {
		msg := errors.Wrap(
			err, "failed to check Task",
		).Error()
		var code int
		switch {
		case users.IsMissing(err):
			code = http.StatusNotFound
		default:
			code = http.StatusInternalServerError
		}
		http.Error(w, msg, code)
		return
	}

	// Store sets the ID and replaces the Task's Notes with
	// Resources.  The user shouldn't see those resource IDs, so
	// they should be cleared after Store.
	notes := tsk.Notes
	err = t.Update(store.Wrap(
		users.CheckUsersExist(allUsers...),
		task.ID(uuid.NewV4()).Store(tsk),
	))
	switch {
	case users.IsMissing(err):
		http.Error(w, errors.Wrap(
			err, "failed to check Task",
		).Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrap(
			err, "failed to store Task",
		).Error(), http.StatusInternalServerError)
		return
	}

	tsk.Notes, tsk.Resources = notes, nil

	notif.Encode(t.Pub, tsk, notif.MakeUserTopic(userID))
	json.NewEncoder(w).Encode(tsk)
}

func (t *Task) Get(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	userID := mw.CtxGetUserID(r)
	tIDString, err := uuid.FromString(ps.ByName("id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	tID := task.ID(tIDString)

	tsk := new(task.Task)
	err = t.View(tID.Load(tsk))
	switch {
	case store.IsMissing(err):
		http.Error(w, "no such task", http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrap(
			err, "failed to find task",
		).Error(), http.StatusInternalServerError)
		return
	case !tsk.Readers[userID] && !tsk.Writers[userID]:
		code := http.StatusUnauthorized
		http.Error(w, http.StatusText(code), code)
		return
	}

	json.NewEncoder(w).Encode(tsk)
}

func (t *Task) Delete(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	userID := mw.CtxGetUserID(r)
	tIDString := ps.ByName("id")
	tUUID, err := uuid.FromString(tIDString)
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	tID := task.ID(tUUID)

	tsk := new(task.Task)
	err = t.View(tID.Load(tsk))
	switch {
	case store.IsMissing(err):
		http.Error(w, "no such task", http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrap(
			err, "failed to find task",
		).Error(), http.StatusInternalServerError)
		return
	case userID != tsk.Owner:
		code := http.StatusUnauthorized
		http.Error(w, http.StatusText(code), code)
		return
	}

	if err := t.Update(tID.Delete); err != nil {
		http.Error(w, errors.Wrap(
			err, "failed to delete task",
		).Error(), http.StatusInternalServerError)
		return
	}

	notif.Encode(t.Pub, task.Deleted(tIDString), notif.MakeUserTopic(userID))
}

func (t *Task) Put(w http.ResponseWriter, r *http.Request, ps htr.Params) {
	userID := mw.CtxGetUserID(r)
	tIDString, err := uuid.FromString(ps.ByName("id"))
	if err != nil {
		http.Error(w, "invalid task ID", http.StatusBadRequest)
		return
	}

	tID := task.ID(tIDString)

	sentTask := new(task.Task)
	if err := json.NewDecoder(r.Body).Decode(sentTask); err != nil {
		http.Error(w, errors.Wrap(
			err, "bad body",
		).Error(), http.StatusBadRequest)
		return
	}

	if sentTask.Owner != userID {
		http.Error(w, errors.Errorf(
			"user %#q cannot update task for user %#q",
			userID, sentTask.Owner,
		).Error(), http.StatusUnauthorized)
		return
	}

	// Make sure the Owner is in the readers / writers.
	sentTask.Readers[userID] = true
	sentTask.Writers[userID] = true

	allUsers := make([]string, len(sentTask.Readers)+len(sentTask.Writers)+1)
	allUsers[0] = userID
	next := 1
	for r := range sentTask.Readers {
		allUsers[next] = r
		next++
	}
	for w := range sentTask.Writers {
		allUsers[next] = w
		next++
	}

	err = t.View(users.CheckUsersExist(allUsers...))
	if err != nil {
		msg := errors.Wrap(
			err, "failed to check Task",
		).Error()
		var code int
		switch {
		case users.IsMissing(err):
			code = http.StatusNotFound
		default:
			code = http.StatusInternalServerError
		}
		http.Error(w, msg, code)
		return
	}

	oldTask := new(task.Task)
	err = t.View(tID.Load(oldTask))
	switch {
	case store.IsMissing(err):
		http.Error(w, "no such task", http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrap(
			err, "failed to find task",
		).Error(), http.StatusInternalServerError)
		return
	case userID != oldTask.Owner:
		code := http.StatusUnauthorized
		http.Error(w, http.StatusText(code), code)
		return
	}

	// Store sets the ID and replaces the Task's Notes with
	// Resources.  The user shouldn't see those resource IDs, so
	// they should be cleared after Store.
	notes := sentTask.Notes
	err = t.Update(store.Wrap(
		task.ID(tID).Store(sentTask),
		users.CheckUsersExist(allUsers...),
	))
	switch {
	case users.IsMissing(err):
		http.Error(w, errors.Wrap(
			err, "failed to check Task",
		).Error(), http.StatusNotFound)
		return
	case err != nil:
		http.Error(w, errors.Wrap(
			err, "failed to store Task",
		).Error(), http.StatusInternalServerError)
		return
	}

	sentTask.ID = tID
	sentTask.Notes = notes
	sentTask.Resources = nil

	notif.Encode(t.Pub, sentTask, notif.MakeUserTopic(userID))
	json.NewEncoder(w).Encode(sentTask)
}

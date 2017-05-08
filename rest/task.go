package rest

import (
	"encoding/json"
	"net/http"
	"net/url"
	"reflect"
	"sort"
	"time"

	"github.com/synapse-garden/sg-proto/notif"
	mw "github.com/synapse-garden/sg-proto/rest/middleware"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream/river"
	"github.com/synapse-garden/sg-proto/task"
	"github.com/synapse-garden/sg-proto/users"
	"github.com/synapse-garden/sg-proto/util"

	"github.com/boltdb/bolt"
	htr "github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

const TaskNotifs = "tasks"

type Task struct {
	*bolt.DB

	river.Pub

	util.Timer
}

func (t *Task) Bind(r *htr.Router) (Cleanup, error) {
	if t.DB == nil {
		return nil, errors.New("nil Task DB handle")
	}

	err := t.Update(func(tx *bolt.Tx) (e error) {
		t.Pub, e = river.NewPub(TaskNotifs, NotifStream, tx)
		return
	})
	if err != nil {
		return nil, err
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

	return t.Cleanup, nil
}

// Cleanup closes the Task's Pub river and deletes it from the DB.
func (t Task) Cleanup() error {
	if err := t.Pub.Close(); err != nil {
		return err
	}

	return t.Update(func(tx *bolt.Tx) error {
		return river.DeletePub(TaskNotifs, NotifStream, tx)
	})
}

func (t Task) GetAll(w http.ResponseWriter, r *http.Request, _ htr.Params) {
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

func (t Task) Create(w http.ResponseWriter, r *http.Request, _ htr.Params) {
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
			"invalid Task: user %#q cannot create task "+
				"for user %#q",
			userID, tsk.Owner,
		).Error(), http.StatusBadRequest)
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

	toUpdate := map[string]struct{}{tsk.Owner: struct{}{}}
	// De-duplicate users to update
	for _, u := range allUsers {
		toUpdate[u] = struct{}{}
	}
	for u := range toUpdate {
		notif.Encode(t.Pub, tsk, notif.MakeUserTopic(u))
	}
	json.NewEncoder(w).Encode(tsk)
}

func (t Task) Get(w http.ResponseWriter, r *http.Request, ps htr.Params) {
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

func (t Task) Delete(w http.ResponseWriter, r *http.Request, ps htr.Params) {
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

	del := task.Deleted(tIDString)
	toUpdate := map[string]bool{tsk.Owner: true}
	// De-duplicate users to update
	for u := range tsk.Readers {
		toUpdate[u] = true
	}
	for u := range tsk.Writers {
		toUpdate[u] = true
	}
	for u := range toUpdate {
		notif.Encode(t.Pub, del, notif.MakeUserTopic(u))
	}
}

func (t Task) Put(w http.ResponseWriter, r *http.Request, ps htr.Params) {
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

	if sentTask.Owner != userID && !sentTask.Writers[userID] {
		http.Error(w, errors.Errorf(
			"invalid task: %#q is not owner or in writers",
			userID,
		).Error(), http.StatusBadRequest)
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
		sentTask.Readers[w] = true
		next++
	}

	err = t.View(users.CheckUsersExist(allUsers...))
	if err != nil {
		msg := errors.Wrap(err, "failed to check Task").Error()
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
	}
	var (
		isOwner  = oldTask.Owner == userID
		isWriter = oldTask.Writers[userID]

		// Non-owners can only modify notes and completion.
		isModifyBounty      = oldTask.Bounty != sentTask.Bounty
		isModifyCompletedBy = oldTask.CompletedBy != sentTask.CompletedBy
		isModifyCompletedAt = func() bool {
			oC := oldTask.CompletedAt
			sC := sentTask.CompletedAt
			if oC != nil && sC != nil {
				return *oC != *sC
			}
			return oC != sC
		}()
		isModifyDueDate = func() bool {
			oD := oldTask.Due
			sD := sentTask.Due
			if oD != nil && sD != nil {
				return *oD != *sD
			}
			return oD != sD
		}()
		isModifyGroup = !reflect.DeepEqual(
			oldTask.Group, sentTask.Group,
		)
		isModifyCompletion = oldTask.Completed != sentTask.Completed
	)
	switch {
	case isModifyCompletedBy && sentTask.Completed:
		http.Error(w, "cannot modify completed-by of complete task", http.StatusBadRequest)
		return
	case isModifyCompletedAt && sentTask.Completed:
		http.Error(w, "cannot modify completed-at of complete task", http.StatusBadRequest)
		return
	case isOwner && isModifyBounty && oldTask.Completed:
		http.Error(w, "cannot modify bounty of complete task", http.StatusBadRequest)
		return
	case isOwner && isModifyDueDate && oldTask.Completed:
		http.Error(w, "cannot modify due date of complete task", http.StatusBadRequest)
		return
	case isOwner && isModifyCompletion && !sentTask.Completed:
		// The owner un-completed the task.  This returns the
		// bounty to the owner and un-sets the completion data.
		t.uncompleteAsOwner(w, allUsers, oldTask, sentTask)
		return
	case isOwner:
		// The owner updated (or completed) the task.  This does
		// not distribute any bounty.
		t.updateAsOwner(w, allUsers, oldTask, sentTask)
		return
	case isModifyBounty, isModifyDueDate, isModifyGroup:
		// Only the owner can do these things.  Unauthorized.
	case isWriter && isModifyCompletion && sentTask.Completed:
		// A writer completed the task.  The bounty is divested
		// from the owner to the writer.  Both are notified.
		t.completeAsWriter(w,
			allUsers, userID,
			oldTask, sentTask,
			t.Now().UTC(),
		)
		return
	case isWriter &&
		isModifyCompletion && !sentTask.Completed &&
		oldTask.CompletedBy == userID:
		// A writer un-completed the task.  Only the person who
		// completed the task can do this.  Owner and completer
		// are notified of bounty changes.
		t.uncompleteAsWriter(w,
			allUsers, userID,
			oldTask, sentTask,
		)
		return
	case isWriter:
		// The writer did something else (like adding notes.)
		t.updateAsWriter(w, allUsers, oldTask, sentTask)
		return
	}
	code := http.StatusUnauthorized
	http.Error(w, http.StatusText(code), code)
	return
}

func (t Task) uncompleteAsOwner(
	w http.ResponseWriter,
	allUsers []string,
	old, new *task.Task,
) {
	notes := new.Notes
	comp := &users.User{Name: old.CompletedBy}
	own := &users.User{Name: old.Owner}
	err := t.Update(store.Wrap(
		old.ID.Store(new),
		users.AddCoin(own, old.Bounty),
		users.AddCoin(comp, -old.Bounty),
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

	new.ID = old.ID
	new.Notes = notes
	new.Resources = nil

	toUpdate := users.DiffGroups(old.Group, new.Group)
	for u, ok := range toUpdate {
		uTopic := notif.MakeUserTopic(u)
		if ok {
			notif.Encode(t.Pub, new, uTopic)
		} else {
			notif.Encode(t.Pub, task.Removed(new.ID), uTopic)
		}
	}
	notif.Encode(t.Pub, own, notif.MakeUserTopic(own.Name))
	notif.Encode(t.Pub, comp, notif.MakeUserTopic(comp.Name))
	json.NewEncoder(w).Encode(new)
}

func (t Task) updateAsOwner(
	w http.ResponseWriter,
	allUsers []string,
	old, new *task.Task,
) {
	notes := new.Notes
	if new.Completed && !old.Completed {
		// The owner completed the task.  Don't divest bounty.
		new.CompletedBy = new.Owner
		now := t.Now()
		new.CompletedAt = &now
	}
	err := t.Update(store.Wrap(
		old.ID.Store(new),
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

	new.ID = old.ID
	new.Notes = notes
	new.Resources = nil

	toUpdate := users.DiffGroups(old.Group, new.Group)
	for u, ok := range toUpdate {
		uTopic := notif.MakeUserTopic(u)
		if ok {
			notif.Encode(t.Pub, new, uTopic)
		} else {
			notif.Encode(t.Pub, task.Removed(new.ID), uTopic)
		}
	}

	json.NewEncoder(w).Encode(new)
}

func (t Task) completeAsWriter(
	w http.ResponseWriter,
	allUsers []string,
	u string,
	old, new *task.Task,
	now time.Time,
) {
	comp := &users.User{Name: u}
	own := &users.User{Name: old.Owner}
	notes := new.Notes
	new.CompletedBy = u
	new.CompletedAt = &now
	err := t.Update(store.Wrap(
		users.CheckUsersExist(allUsers...),
		users.AddCoin(own, -new.Bounty),
		users.AddCoin(comp, new.Bounty),
		old.ID.Store(new),
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

	new.ID = old.ID
	new.Notes = notes
	new.Resources = nil

	for up := range users.AllUsers(new.Group) {
		// Notify each user of the change.
		notif.Encode(t.Pub, new, notif.MakeUserTopic(up))
	}
	// Notify the owner that the bounty was divested
	notif.Encode(t.Pub, own, notif.MakeUserTopic(own.Name))
	// Notify the completing writer of his profile bounty update
	notif.Encode(t.Pub, comp, notif.MakeUserTopic(comp.Name))
	json.NewEncoder(w).Encode(new)
}

func (t Task) uncompleteAsWriter(
	w http.ResponseWriter,
	allUsers []string,
	u string,
	old, new *task.Task,
) {
	comp := &users.User{Name: u}
	own := &users.User{Name: old.Owner}

	notes := new.Notes
	new.CompletedBy = ""
	new.CompletedAt = nil
	err := t.Update(store.Wrap(
		users.CheckUsersExist(allUsers...),
		users.AddCoin(own, new.Bounty),
		users.AddCoin(comp, -new.Bounty),
		old.ID.Store(new),
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

	new.ID = old.ID
	new.Notes = notes
	new.Resources = nil

	for up := range users.AllUsers(new.Group) {
		// Notify each user of the change
		notif.Encode(t.Pub, new, notif.MakeUserTopic(up))
	}
	// Notify the completing writer of his profile bounty update
	notif.Encode(t.Pub, comp, notif.MakeUserTopic(comp.Name))
	// Notify the owner his bounty was returned
	notif.Encode(t.Pub, own, notif.MakeUserTopic(own.Name))
	json.NewEncoder(w).Encode(new)
}

func (t Task) updateAsWriter(
	w http.ResponseWriter,
	allUsers []string,
	old, new *task.Task,
) {
	notes := new.Notes
	err := t.Update(store.Wrap(
		users.CheckUsersExist(allUsers...),
		old.ID.Store(new),
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

	new.ID = old.ID
	new.Notes = notes
	new.Resources = nil

	for up := range users.AllUsers(new.Group) {
		// Notify each user of the change
		notif.Encode(t.Pub, new, notif.MakeUserTopic(up))
	}
	json.NewEncoder(w).Encode(new)
}

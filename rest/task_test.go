package rest_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	htt "net/http/httptest"
	"reflect"
	"time"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/task"
	sgt "github.com/synapse-garden/sg-proto/testing"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/davecgh/go-spew/spew"
	htr "github.com/julienschmidt/httprouter"
	uuid "github.com/satori/go.uuid"
	ws "golang.org/x/net/websocket"
	. "gopkg.in/check.v1"
)

var _ = rest.API(new(rest.Task))

func prepTaskAPI(c *C,
	r *htr.Router,
	api *rest.Task,
	names ...string,
) (*htt.Server, rest.Cleanup, map[string]auth.Token) {
	tokens := make(map[string]auth.Token)

	for _, user := range names {
		_, err := sgt.MakeLogin(user, "some-password", api.DB)
		c.Assert(err, IsNil)
		sesh := new(auth.Session)
		c.Assert(sgt.GetSession(user, sesh, api.DB), IsNil)
		tokens[user] = sesh.Token
	}

	cc, err := api.Bind(r)
	c.Assert(err, IsNil)

	// Make a testing server to run it.
	return htt.NewServer(r), cc, tokens
}

func (s *RESTSuite) TestTaskBind(c *C) {
	_, err := new(rest.Task).Bind(nil)
	c.Assert(err, ErrorMatches, "nil Task DB handle")

	var (
		r = htr.New()

		now       = time.Now().UTC()
		someWhen  = now.Add(2 * time.Hour)
		beforeNow = now.Add(-1 * time.Hour)

		_, notifErr = rest.Notif{DB: s.db}.Bind(r)
		api         = &rest.Task{DB: s.db, Timer: sgt.Timer(now)}

		srv, cc, tokens = prepTaskAPI(c, r, api, "bodie", "bob")
	)

	defer srv.Close()
	defer cc()
	c.Assert(notifErr, IsNil)

	// Get websocket connection for "bodie".
	connBodie, err := sgt.GetWSClient(
		base64.RawURLEncoding.EncodeToString(tokens["bodie"]),
		srv.URL+"/notifs",
	)
	c.Assert(err, IsNil)

	// Get websocket connection for "bob".
	connBob, err := sgt.GetWSClient(
		base64.RawURLEncoding.EncodeToString(tokens["bob"]),
		srv.URL+"/notifs",
	)
	c.Assert(err, IsNil)

	t := &task.Task{
		Group: users.Group{
			Owner: "bodie",

			Readers: map[string]bool{"bodie": true},
			Writers: map[string]bool{"bodie": true},
		},

		Name: "V. Important Task",

		Notes: []string{"hello world", "goodbye world"},

		Due: &someWhen,
	}
	tOverdue := &task.Task{
		Group: users.Group{
			Owner: "bodie",

			Readers: map[string]bool{"bodie": true},
			Writers: map[string]bool{"bodie": true},
		},

		Name: "V. Old Task",

		Notes: []string{"oops", "big trouble"},

		Due: &beforeNow,
	}
	tNoDue := &task.Task{
		Group: users.Group{
			Owner: "bodie",

			Readers: map[string]bool{"bodie": true},
			Writers: map[string]bool{"bodie": true},
		},

		Name: "V. Indefinite Task",

		Notes: []string{"nobody will ever do this"},
	}
	tComplete := &task.Task{
		Group: users.Group{
			Owner: "bodie",

			Readers: map[string]bool{"bodie": true},
			Writers: map[string]bool{"bodie": true},
		},

		Name: "V. Finished Task",

		Notes: []string{"this is done", "cool", "and nice"},

		Due:       &now,
		Completed: true,
	}

	c.Log("POST new task")
	send, err := json.Marshal(t)
	c.Assert(err, IsNil)
	sendOld, err := json.Marshal(tOverdue)
	c.Assert(err, IsNil)
	sendDone, err := json.Marshal(tComplete)
	c.Assert(err, IsNil)
	sendNoDue, err := json.Marshal(tNoDue)
	c.Assert(err, IsNil)

	req := htt.NewRequest("POST", "/tasks", bytes.NewBuffer(send))
	req.Header = sgt.Bearer(tokens["bodie"])
	oldReq := htt.NewRequest("POST", "/tasks", bytes.NewBuffer(sendOld))
	oldReq.Header = sgt.Bearer(tokens["bodie"])
	doneReq := htt.NewRequest("POST", "/tasks", bytes.NewBuffer(sendDone))
	doneReq.Header = sgt.Bearer(tokens["bodie"])
	noDueReq := htt.NewRequest("POST", "/tasks", bytes.NewBuffer(sendNoDue))
	noDueReq.Header = sgt.Bearer(tokens["bodie"])

	w := htt.NewRecorder()
	r.ServeHTTP(w, req)
	c.Check(w.Code, Equals, http.StatusOK)
	got := new(task.Task)
	c.Assert(json.Unmarshal(w.Body.Bytes(), got), IsNil)
	c.Check(got.Owner, Equals, t.Owner)
	c.Check(got.Writers, DeepEquals, t.Writers)
	c.Check(got.Readers, DeepEquals, t.Readers)
	c.Check(got.Resources, IsNil)
	c.Check(got.Notes, DeepEquals, t.Notes)
	c.Check(got.Due.Equal(*t.Due), Equals, true)

	notif := new(store.ResourceBox)
	c.Assert(ws.JSON.Receive(connBodie, notif), IsNil)
	c.Check(notif, DeepEquals, &store.ResourceBox{
		Name: "tasks",
		Contents: map[string]interface{}{
			"owner":   "bodie",
			"readers": map[string]interface{}{"bodie": true},
			"writers": map[string]interface{}{"bodie": true},
			"id":      uuid.UUID(got.ID).String(),
			"name":    "V. Important Task",
			"notes": []interface{}{
				"hello world",
				"goodbye world",
			},
			"due":       someWhen.Format(time.RFC3339Nano),
			"completed": false,
		},
	})

	x := new(map[string]interface{})

	w = htt.NewRecorder()
	r.ServeHTTP(w, oldReq)
	c.Check(w.Code, Equals, http.StatusOK)
	gotOld := new(task.Task)
	c.Assert(json.Unmarshal(w.Body.Bytes(), gotOld), IsNil)
	c.Assert(ws.JSON.Receive(connBodie, x), IsNil)

	w = htt.NewRecorder()
	r.ServeHTTP(w, noDueReq)
	c.Check(w.Code, Equals, http.StatusOK)
	gotNoDue := new(task.Task)
	c.Assert(json.Unmarshal(w.Body.Bytes(), gotNoDue), IsNil)
	c.Assert(ws.JSON.Receive(connBodie, x), IsNil)

	w = htt.NewRecorder()
	r.ServeHTTP(w, doneReq)
	c.Check(w.Code, Equals, http.StatusOK)
	gotDone := new(task.Task)
	c.Assert(json.Unmarshal(w.Body.Bytes(), gotDone), IsNil)
	c.Assert(ws.JSON.Receive(connBodie, x), IsNil)

	newGot := new(task.Task)

	otherT := new(task.Task)
	*otherT = *t
	otherT.ID = got.ID
	otherT.Bounty = 5
	otherT.Notes = append(otherT.Notes, "something else")
	otherT.Readers = map[string]bool{"bodie": true, "bob": true}
	otherT.Writers = map[string]bool{"bodie": true, "bob": true}

	doneGot := new(task.Task)
	*doneGot = *otherT
	doneGot.Completed = true
	doneGot.Notes = append(doneGot.Notes, "another thing entirely")
	doneDone := new(task.Task)
	*doneDone = *doneGot
	doneDone.CompletedBy = "bob"
	doneDone.CompletedAt = new(time.Time)
	*doneDone.CompletedAt = now.UTC()

	badUsersT := new(task.Task)
	*badUsersT = *t
	badUsersT.Readers = map[string]bool{
		"bodie":   true,
		"floobob": true,
	}
	badUsersT.Writers = map[string]bool{
		"bodie":   true,
		"floobob": true,
	}

	c.Logf("POST succeeded: new ID [% x]", got.ID)

	c.Log("Should POST new task with two users and notify both, then delete")
	sendMultiNotif := &task.Task{
		Group: users.Group{
			Owner:   "bodie",
			Readers: map[string]bool{"bodie": true, "bob": true},
			Writers: map[string]bool{"bodie": true, "bob": true},
		},
		Name: "a test",
	}
	sendMultiNotifBs, err := json.Marshal(sendMultiNotif)
	c.Assert(err, IsNil)
	multiNotifReq := htt.NewRequest("POST", "/tasks", bytes.NewBuffer(sendMultiNotifBs))
	multiNotifReq.Header = sgt.Bearer(tokens["bodie"])

	w = htt.NewRecorder()
	r.ServeHTTP(w, multiNotifReq)
	c.Check(w.Code, Equals, http.StatusOK)
	multiNotifGot := new(task.Task)
	c.Assert(json.Unmarshal(w.Body.Bytes(), multiNotifGot), IsNil)
	c.Assert(uuid.Equal(uuid.UUID(multiNotifGot.ID), uuid.Nil), Equals, false)
	sendMultiNotif.ID = multiNotifGot.ID
	c.Check(multiNotifGot, DeepEquals, sendMultiNotif)
	c.Assert(ws.JSON.Receive(connBodie, x), IsNil)
	c.Check((*x)["name"], Equals, "tasks")
	c.Assert(ws.JSON.Receive(connBob, x), IsNil)
	c.Check((*x)["name"], Equals, "tasks")
	delReq := htt.NewRequest("DELETE", "/tasks/"+uuid.UUID(multiNotifGot.ID).String(), nil)
	delReq.Header = sgt.Bearer(tokens["bodie"])
	w = htt.NewRecorder()
	r.ServeHTTP(w, delReq)
	c.Check(w.Code, Equals, http.StatusOK)
	c.Assert(ws.JSON.Receive(connBodie, x), IsNil)
	c.Check((*x)["name"], Equals, "task-deleted")
	c.Assert(ws.JSON.Receive(connBob, x), IsNil)
	c.Check((*x)["name"], Equals, "task-deleted")

	for i, test := range []struct {
		should string

		verb, path       string
		header           http.Header
		body             interface{}
		expectStatus     int
		into, expectResp interface{}

		expectNotifs  map[*ws.Conn][]*store.ResourceBox
		expectHeaders []http.Header
	}{{
		should: "return [] when none exist for user",
		verb:   "GET", path: "/tasks",
		header:       sgt.Bearer(tokens["bob"]),
		expectStatus: http.StatusOK,
		into:         new([]task.Task),
		expectResp:   new([]task.Task),
	}, {
		should: "error on bad query params",
		verb:   "GET", path: "/tasks?%q",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp: "bad query parameters: invalid URL " +
			`escape "%q"` + "\n",
	}, {
		should: "error on unknown query param",
		verb:   "GET", path: "/tasks?foo=bar",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp:   `unknown query parameter "foo"` + "\n",
	}, {
		should: `error on bad "overdue" param value`,
		verb:   "GET", path: "/tasks?overdue=1",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp: `bad value "1" for query parameter ` +
			`"overdue"` + "\n",
	}, {
		should: `error on bad "complete" param value`,
		verb:   "GET", path: "/tasks?complete=1",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp: `bad value "1" for query parameter ` +
			`"complete"` + "\n",
	}, {
		should: `return only complete tasks`,
		verb:   "GET", path: "/tasks?complete=true",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusOK,
		into:         new([]task.Task),
		expectResp:   &[]task.Task{*gotDone},
	}, {
		should: `return only overdue tasks`,
		verb:   "GET", path: "/tasks?overdue=true",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusOK,
		into:         new([]task.Task),
		expectResp:   &[]task.Task{*gotOld},
	}, {
		should: "return all tasks which user has access to",
		verb:   "GET", path: "/tasks",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusOK,
		into:         new([]task.Task),
		expectResp: &[]task.Task{
			*gotOld, *gotDone,
			*got, *gotNoDue,
		},
	}, {
		should: "return error for bad task body",
		verb:   "POST", path: "/tasks",
		header:       sgt.Bearer(tokens["bob"]),
		expectStatus: http.StatusBadRequest,
		body:         "oops boops",
		into:         new(string),
		expectResp: "bad body: " +
			"json: cannot unmarshal string into Go value " +
			"of type task.Task\n",
	}, {
		should: "forbid creation of tasks for wrong users",
		verb:   "POST", path: "/tasks",
		header:       sgt.Bearer(tokens["bob"]),
		expectStatus: http.StatusBadRequest,
		body:         t,
		into:         new(string),
		expectResp: "invalid Task: user `bob` cannot create " +
			"task for user `bodie`\n",
	}, {
		should: "forbid creation of tasks with nonexistent users",
		verb:   "POST", path: "/tasks",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusNotFound,
		body:         badUsersT,
		into:         new(string),
		expectResp: "failed to check Task: user `floobob` " +
			"not found\n",
	}, {
		should: "reject bad task IDs",
		verb:   "PUT", path: "/tasks/boop",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		body:         t,
		into:         new(string),
		expectResp:   "invalid task ID\n",
	}, {
		should:       "reject bad body",
		verb:         "PUT",
		path:         "/tasks/" + uuid.UUID(got.ID).String(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		body:         "woops",
		into:         new(string),
		expectResp: "bad body: json: cannot unmarshal string " +
			"into Go value of type task.Task\n",
	}, {
		should: "reject invalid PUT updates (user not owner " +
			"or writer)",
		verb:         "PUT",
		path:         "/tasks/" + uuid.UUID(got.ID).String(),
		header:       sgt.Bearer(tokens["bob"]),
		expectStatus: http.StatusBadRequest,
		body:         badUsersT,
		into:         new(string),
		expectResp: "invalid task: `bob` is not owner or in " +
			"writers\n",
	}, {
		should:       "forbid update of tasks with nonexistent users",
		verb:         "PUT",
		path:         "/tasks/" + uuid.UUID(got.ID).String(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusNotFound,
		body:         badUsersT,
		into:         new(string),
		expectResp: "failed to check Task: user `floobob` " +
			"not found\n",
	}, {
		should: "return 404 for nonexistent tasks",
		verb:   "PUT", path: "/tasks/" + uuid.Nil.String(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusNotFound,
		body:         t,
		into:         new(string),
		expectResp:   "no such task\n",
	}, {
		should:       "reject unauthorized update",
		verb:         "PUT",
		path:         "/tasks/" + uuid.UUID(got.ID).String(),
		header:       sgt.Bearer(tokens["bob"]),
		expectStatus: http.StatusUnauthorized,
		body:         otherT,
		into:         new(string),
		expectResp:   "Unauthorized\n",
	}, {
		should:       "update a task as expected",
		verb:         "PUT",
		path:         "/tasks/" + uuid.UUID(got.ID).String(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusOK,
		body:         otherT,
		into:         newGot,
		expectResp:   otherT,
		expectNotifs: map[*ws.Conn][]*store.ResourceBox{connBob: {{
			Name: "tasks",
			Contents: map[string]interface{}{
				"owner": "bodie",
				"readers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"writers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"id":   uuid.UUID(got.ID).String(),
				"name": "V. Important Task",
				"notes": []interface{}{
					"hello world",
					"goodbye world",
					"something else",
				},
				"bounty":    float64(5),
				"due":       someWhen.Format(time.RFC3339Nano),
				"completed": false,
			},
		}}, connBodie: {{
			Name: "tasks",
			Contents: map[string]interface{}{
				"owner": "bodie",
				"readers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"writers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"id":   uuid.UUID(got.ID).String(),
				"name": "V. Important Task",
				"notes": []interface{}{
					"hello world",
					"goodbye world",
					"something else",
				},
				"bounty":    float64(5),
				"due":       someWhen.Format(time.RFC3339Nano),
				"completed": false,
			},
		}}},
	}, {
		should:       "complete a task with a bounty as expected",
		verb:         "PUT",
		path:         "/tasks/" + uuid.UUID(doneGot.ID).String(),
		header:       sgt.Bearer(tokens["bob"]),
		expectStatus: http.StatusOK,
		body:         doneGot,
		into:         new(task.Task),
		expectResp:   doneDone,
		expectNotifs: map[*ws.Conn][]*store.ResourceBox{connBodie: {{
			Name: "tasks",
			Contents: map[string]interface{}{
				"owner":  "bodie",
				"bounty": float64(5),
				"readers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"writers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"id":   uuid.UUID(got.ID).String(),
				"name": "V. Important Task",
				"notes": []interface{}{
					"hello world",
					"goodbye world",
					"something else",
					"another thing entirely",
				},
				"due":         someWhen.Format(time.RFC3339Nano),
				"completed":   true,
				"completedBy": "bob",
				"completedAt": now.UTC().Format(time.RFC3339Nano),
			},
		}, {
			Name: "users",
			Contents: map[string]interface{}{
				"name": "bodie",
				"coin": float64(-5),
			},
		}}, connBob: {{
			Name: "tasks",
			Contents: map[string]interface{}{
				"owner":  "bodie",
				"bounty": float64(5),
				"readers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"writers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"id":   uuid.UUID(got.ID).String(),
				"name": "V. Important Task",
				"notes": []interface{}{
					"hello world",
					"goodbye world",
					"something else",
					"another thing entirely",
				},
				"due":         someWhen.Format(time.RFC3339Nano),
				"completed":   true,
				"completedBy": "bob",
				"completedAt": now.UTC().Format(time.RFC3339Nano),
			},
		}, {
			Name: "users",
			Contents: map[string]interface{}{
				"name": "bob",
				"coin": float64(5),
			},
		}}},
	}, {
		should: "not let anyone update bounty of completed tasks",
		verb:   "PUT",
		path:   "/tasks/" + uuid.UUID(got.ID).String(),
		body: func() *task.Task {
			t := *doneDone
			t.Bounty = 3
			return &t
		}(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp:   "cannot modify bounty of complete task\n",
	}, {
		should: "not let anyone update due date of completed tasks",
		verb:   "PUT",
		path:   "/tasks/" + uuid.UUID(got.ID).String(),
		body: func() *task.Task {
			t := *doneDone
			t.Due = &beforeNow
			return &t
		}(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp:   "cannot modify due date of complete task\n",
	}, {
		should: "not let anyone update completed-by of completed tasks",
		verb:   "PUT",
		path:   "/tasks/" + uuid.UUID(got.ID).String(),
		body: func() *task.Task {
			t := *doneDone
			t.CompletedBy = ""
			return &t
		}(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp:   "cannot modify completed-by of complete task\n",
	}, {
		should: "not let anyone update completed-at of completed tasks",
		verb:   "PUT",
		path:   "/tasks/" + uuid.UUID(got.ID).String(),
		body: func() *task.Task {
			t := *doneDone
			t.CompletedAt = &someWhen
			return &t
		}(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp:   "cannot modify completed-at of complete task\n",
	}, {
		should: "not let anyone update due date of completed tasks",
		verb:   "PUT",
		path:   "/tasks/" + uuid.UUID(got.ID).String(),
		body: func() *task.Task {
			t := *doneDone
			t.Due = &beforeNow
			return &t
		}(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp:   "cannot modify due date of complete task\n",
	}, {
		should:       "get updated task as expected",
		verb:         "GET",
		path:         "/tasks/" + uuid.UUID(got.ID).String(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusOK,
		into:         new(task.Task),
		expectResp:   doneDone,
	}, {
		should:       "uncomplete a task with a bounty as expected",
		verb:         "PUT",
		path:         "/tasks/" + uuid.UUID(doneGot.ID).String(),
		header:       sgt.Bearer(tokens["bob"]),
		expectStatus: http.StatusOK,
		body:         otherT,
		into:         new(task.Task),
		expectResp:   otherT,
		expectNotifs: map[*ws.Conn][]*store.ResourceBox{connBodie: {{
			Name: "tasks",
			Contents: map[string]interface{}{
				"owner":  "bodie",
				"bounty": float64(5),
				"readers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"writers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"id":   uuid.UUID(got.ID).String(),
				"name": "V. Important Task",
				"notes": []interface{}{
					"hello world",
					"goodbye world",
					"something else",
				},
				"due":       someWhen.Format(time.RFC3339Nano),
				"completed": false,
			},
		}, {
			Name: "users",
			Contents: map[string]interface{}{
				"name": "bodie",
				"coin": float64(0),
			},
		}}, connBob: {{
			Name: "tasks",
			Contents: map[string]interface{}{
				"owner":  "bodie",
				"bounty": float64(5),
				"readers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"writers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"id":   uuid.UUID(got.ID).String(),
				"name": "V. Important Task",
				"notes": []interface{}{
					"hello world",
					"goodbye world",
					"something else",
				},
				"due":       someWhen.Format(time.RFC3339Nano),
				"completed": false,
			},
		}, {
			Name: "users",
			Contents: map[string]interface{}{
				"name": "bob",
				"coin": float64(0),
			},
		}}},
	}, {
		should:       "update a task as a writer normally",
		verb:         "PUT",
		path:         "/tasks/" + uuid.UUID(doneGot.ID).String(),
		header:       sgt.Bearer(tokens["bob"]),
		expectStatus: http.StatusOK,
		body: func() *task.Task {
			tt := *otherT
			tt.Notes = make([]string, len(tt.Notes))
			copy(tt.Notes, otherT.Notes)
			tt.Notes = append(tt.Notes, "boopy doopy")
			return &tt
		}(),
		into: new(task.Task),
		expectResp: func() *task.Task {
			tt := *otherT
			tt.Notes = make([]string, len(tt.Notes))
			copy(tt.Notes, otherT.Notes)
			tt.Notes = append(tt.Notes, "boopy doopy")
			return &tt
		}(),
		expectNotifs: map[*ws.Conn][]*store.ResourceBox{connBodie: {{
			Name: "tasks",
			Contents: map[string]interface{}{
				"owner":  "bodie",
				"bounty": float64(5),
				"readers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"writers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"id":   uuid.UUID(got.ID).String(),
				"name": "V. Important Task",
				"notes": []interface{}{
					"hello world",
					"goodbye world",
					"something else",
					"boopy doopy",
				},
				"due":       someWhen.Format(time.RFC3339Nano),
				"completed": false,
			},
		}}, connBob: {{
			Name: "tasks",
			Contents: map[string]interface{}{
				"owner":  "bodie",
				"bounty": float64(5),
				"readers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"writers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"id":   uuid.UUID(got.ID).String(),
				"name": "V. Important Task",
				"notes": []interface{}{
					"hello world",
					"goodbye world",
					"something else",
					"boopy doopy",
				},
				"due":       someWhen.Format(time.RFC3339Nano),
				"completed": false,
			},
		}}},
	}, {
		should:       "complete a task as an owner normally",
		verb:         "PUT",
		path:         "/tasks/" + uuid.UUID(doneGot.ID).String(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusOK,
		body: func() *task.Task {
			tt := *otherT
			tt.Completed = true
			return &tt
		}(),
		into: new(task.Task),
		expectResp: func() *task.Task {
			tt := *otherT
			tt.Completed = true
			tt.CompletedAt = &now
			tt.CompletedBy = "bodie"
			return &tt
		}(),
		expectNotifs: map[*ws.Conn][]*store.ResourceBox{connBodie: {{
			Name: "tasks",
			Contents: map[string]interface{}{
				"owner":  "bodie",
				"bounty": float64(5),
				"readers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"writers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"id":   uuid.UUID(got.ID).String(),
				"name": "V. Important Task",
				"notes": []interface{}{
					"hello world",
					"goodbye world",
					"something else",
				},
				"due":         someWhen.Format(time.RFC3339Nano),
				"completed":   true,
				"completedBy": "bodie",
				"completedAt": now.Format(time.RFC3339Nano),
			},
		}}, connBob: {{
			Name: "tasks",
			Contents: map[string]interface{}{
				"owner":  "bodie",
				"bounty": float64(5),
				"readers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"writers": map[string]interface{}{
					"bodie": true,
					"bob":   true,
				},
				"id":   uuid.UUID(got.ID).String(),
				"name": "V. Important Task",
				"notes": []interface{}{
					"hello world",
					"goodbye world",
					"something else",
				},
				"due":         someWhen.Format(time.RFC3339Nano),
				"completed":   true,
				"completedBy": "bodie",
				"completedAt": now.Format(time.RFC3339Nano),
			},
		}}},
	}, {
		should:       "return error on bad task ID",
		verb:         "DELETE",
		path:         "/tasks/boop",
		header:       sgt.Bearer(tokens["bob"]),
		expectStatus: http.StatusBadRequest,
		into:         new(string),
		expectResp:   "invalid task ID\n",
	}, {
		should:       "forbid deletion of tasks by non-owners",
		verb:         "DELETE",
		path:         "/tasks/" + uuid.UUID(got.ID).String(),
		header:       sgt.Bearer(tokens["bob"]),
		expectStatus: http.StatusUnauthorized,
		into:         new(string),
		expectResp:   "Unauthorized\n",
	}, {
		should:       "error on nonexistent tasks",
		verb:         "DELETE",
		path:         "/tasks/" + uuid.Nil.String(),
		header:       sgt.Bearer(tokens["bob"]),
		expectStatus: http.StatusNotFound,
		into:         new(string),
		expectResp:   "no such task\n",
	}, {
		should:       "delete correct tasks normally",
		verb:         "DELETE",
		path:         "/tasks/" + uuid.UUID(got.ID).String(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusOK,
		into:         new(string),
		expectResp:   "",
		expectNotifs: map[*ws.Conn][]*store.ResourceBox{connBodie: {{
			Name:     "task-deleted",
			Contents: uuid.UUID(got.ID).String(),
		}}, connBob: {{
			Name:     "task-deleted",
			Contents: uuid.UUID(got.ID).String(),
		}}},
		expectHeaders: []http.Header{},
	}, {
		should:       "not get deleted tasks",
		verb:         "GET",
		path:         "/tasks/" + uuid.UUID(got.ID).String(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusNotFound,
		into:         new(string),
		expectResp:   "no such task\n",
	}} {
		c.Logf("test %d: %s on %s should %s", i,
			test.verb, test.path,
			test.should,
		)
		c.Assert(sgt.ExpectResponse(r,
			test.path, test.verb,
			test.body,
			test.into, test.expectResp,
			test.expectStatus,
			test.header,
			test.expectHeaders...,
		), IsNil)

		for conn, expects := range test.expectNotifs {
			// Check that the given user's notif conn
			// received the expected notifs.

			// ns starts as a copy of expected notifs, and
			// each is removed from 'ns' when it is found;
			// ns should be empty at the end of the loop.
			ns := expects
			var saw []*store.ResourceBox
			for _ = range expects {
				// Note this will not account for
				// unexpected notifs, but they will
				// cause problems later...
				notif := new(store.ResourceBox)
				c.Assert(ws.JSON.Receive(conn, notif), IsNil)
				saw = append(saw, notif)
			find:
				for j, n := range ns {
					// Was that one of the expected notifs?
					if reflect.DeepEqual(notif, n) {
						// If found, remove from expected
						// by re-slicing and stop search.
						ns = append(ns[:j], ns[j+1:]...)
						break find
					}
				}
			}

			c.Check(len(ns), Equals, 0)
			if c.Failed() {
				c.Logf("expected notifs: %s\n"+
					"  but never saw %s\n"+
					"  got %s\n",
					spew.Sdump(expects),
					spew.Sdump(ns),
					spew.Sdump(saw),
				)
			}
		}
	}

	c.Assert(connBodie.Close(), IsNil)
	c.Assert(connBob.Close(), IsNil)
}

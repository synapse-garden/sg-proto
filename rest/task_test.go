package rest_test

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	htt "net/http/httptest"
	"time"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream/river"
	"github.com/synapse-garden/sg-proto/task"
	sgt "github.com/synapse-garden/sg-proto/testing"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
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
) (*htt.Server, map[string]auth.Token) {
	tokens := make(map[string]auth.Token)

	for _, user := range names {
		_, err := sgt.MakeLogin(user, "some-password", api.DB)
		c.Assert(err, IsNil)
		sesh := new(auth.Session)
		c.Assert(sgt.GetSession(user, sesh, api.DB), IsNil)
		tokens[user] = sesh.Token
	}

	c.Assert(api.Bind(r), IsNil)

	// Make a testing server to run it.
	return htt.NewServer(r), tokens
}

func cleanupTaskAPI(c *C, api *rest.Task) {
	c.Assert(api.Pub.Close(), IsNil)
	c.Assert(api.Update(func(tx *bolt.Tx) error {
		return river.DeletePub(rest.TaskNotifs, rest.NotifStream, tx)
	}), IsNil)
}

func (s *RESTSuite) TestTaskBind(c *C) {
	c.Assert(
		new(rest.Task).Bind(htr.New()),
		ErrorMatches,
		"Bind called with nil DB handle",
	)

	var (
		r           = htr.New()
		notifErr    = rest.Notif{DB: s.db}.Bind(r)
		api         = &rest.Task{DB: s.db}
		srv, tokens = prepTaskAPI(c, r, api, "bodie", "bob")
		now         = time.Now()
		someWhen    = now.Add(2 * time.Hour)
		beforeNow   = now.Add(-1 * time.Hour)
	)

	defer srv.Close()
	c.Assert(notifErr, IsNil)

	// Get websocket connection for "bodie".
	conn, err := sgt.GetWSClient(
		base64.RawURLEncoding.EncodeToString(tokens["bodie"]),
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
	c.Assert(ws.JSON.Receive(conn, notif), IsNil)
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
	c.Assert(ws.JSON.Receive(conn, x), IsNil)

	w = htt.NewRecorder()
	r.ServeHTTP(w, noDueReq)
	c.Check(w.Code, Equals, http.StatusOK)
	gotNoDue := new(task.Task)
	c.Assert(json.Unmarshal(w.Body.Bytes(), gotNoDue), IsNil)
	c.Assert(ws.JSON.Receive(conn, x), IsNil)

	w = htt.NewRecorder()
	r.ServeHTTP(w, doneReq)
	c.Check(w.Code, Equals, http.StatusOK)
	gotDone := new(task.Task)
	c.Assert(json.Unmarshal(w.Body.Bytes(), gotDone), IsNil)
	c.Assert(ws.JSON.Receive(conn, x), IsNil)

	newGot := new(task.Task)
	otherT := new(task.Task)
	*otherT = *t
	otherT.ID = got.ID
	otherT.Notes = append(otherT.Notes, "something else")
	otherT.Readers = map[string]bool{"bodie": true}
	otherT.Writers = map[string]bool{"bodie": true}

	badUsersT := new(task.Task)
	*badUsersT = *t
	badUsersT.Readers = map[string]bool{"bodie": true, "floobob": true}
	badUsersT.Writers = map[string]bool{"bodie": true, "floobob": true}

	c.Logf("POST succeeded: new ID [% x]", got.ID)

	for i, test := range []struct {
		should string

		verb, path       string
		header           http.Header
		body             interface{}
		expectStatus     int
		into, expectResp interface{}

		expectNotif *store.ResourceBox
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
		expectStatus: http.StatusUnauthorized,
		body:         t,
		into:         new(string),
		expectResp: "user `bob` cannot create task for user " +
			"`bodie`\n",
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
		should: "reject bad PUT body",
		verb:   "PUT", path: "/tasks/" + uuid.UUID(got.ID).String(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		body:         "woops",
		into:         new(string),
		expectResp: "bad body: json: cannot unmarshal string " +
			"into Go value of type task.Task\n",
	}, {
		should: "reject unauthorized task owner",
		verb:   "PUT", path: "/tasks/boop",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusBadRequest,
		body:         t,
		into:         new(string),
		expectResp:   "invalid task ID\n",
	}, {
		should: "return 404 for nonexistent tasks",
		verb:   "PUT", path: "/tasks/" + uuid.Nil.String(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusNotFound,
		body:         t,
		into:         new(string),
		expectResp:   "no such task\n",
	}, {
		should: "forbid update of tasks with nonexistent users",
		verb:   "POST", path: "/tasks",
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusNotFound,
		body:         badUsersT,
		into:         new(string),
		expectResp: "failed to check Task: user `floobob` " +
			"not found\n",
	}, {
		should: "reject unauthorized update",
		verb:   "PUT", path: "/tasks/" + uuid.UUID(got.ID).String(),
		header:       sgt.Bearer(tokens["bob"]),
		expectStatus: http.StatusUnauthorized,
		body:         otherT,
		into:         new(string),
		expectResp: "user `bob` cannot update task for user " +
			"`bodie`\n",
	}, {
		should: "update a task as expected",
		verb:   "PUT", path: "/tasks/" + uuid.UUID(got.ID).String(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusOK,
		body:         otherT,
		into:         newGot,
		expectResp:   otherT,
		expectNotif: &store.ResourceBox{
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
					"something else",
				},
				"due":       someWhen.Format(time.RFC3339Nano),
				"completed": false,
			},
		},
	}, {
		should: "get updated task as expected",
		verb:   "GET", path: "/tasks/" + uuid.UUID(got.ID).String(),
		header:       sgt.Bearer(tokens["bodie"]),
		expectStatus: http.StatusOK,
		into:         new(task.Task),
		expectResp:   otherT,
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
		expectNotif: &store.ResourceBox{
			Name:     "task-deleted",
			Contents: uuid.UUID(got.ID).String(),
		},
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
		), IsNil)

		if e := test.expectNotif; e != nil {
			notif := new(store.ResourceBox)
			c.Assert(ws.JSON.Receive(conn, notif), IsNil)
			c.Check(notif, DeepEquals, e)
		}
	}

	c.Assert(conn.Close(), IsNil)

	cleanupTaskAPI(c, api)
}

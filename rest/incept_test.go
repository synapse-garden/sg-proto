package rest_test

import (
	"bytes"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	htt "net/http/httptest"
	"strings"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	"github.com/julienschmidt/httprouter"
	. "gopkg.in/check.v1"
)

func (s *RESTSuite) TestIncept(c *C) {
	correctURL := "/incept/"

	correctSha := sha256.Sum256([]byte("hello"))
	correctHashB64 := base64.StdEncoding.EncodeToString(correctSha[:])

	correctBodyBs, err := json.Marshal(&auth.Login{
		User: users.User{
			Name: "some-name",
		},
		PWHash: correctSha[:],
	})
	c.Assert(err, IsNil)
	correctBody := string(correctBodyBs)

	tkts := make([]string, len(s.tickets))
	for i, t := range s.tickets {
		tkts[i] = t.String()
	}
	c.Logf("correct tickets:\n  %+v", strings.Join(tkts, "\n  "))
	c.Logf("correct body: %#q", correctBody)
	c.Logf(`correct hash of "hello": %#q`, correctHashB64)

	for i, test := range []struct {
		method, url, body string
		expectCode        int
		expectBody        string
		expectUser        string
	}{{
		method:     "GET",
		url:        correctURL + tkts[0],
		body:       correctBody,
		expectCode: http.StatusMethodNotAllowed,
		expectBody: "Method Not Allowed\n",
	}, {
		method:     "POST",
		url:        correctURL + tkts[0],
		body:       `{{"name":"bob"}`,
		expectCode: http.StatusBadRequest,
		expectBody: "failed to decode: invalid character '{' " +
			"looking for beginning of object key string\n",
	}, {
		method:     "POST",
		url:        correctURL + tkts[0],
		body:       `{"name":"bob","pwhash":"12345"}`,
		expectCode: http.StatusBadRequest,
		expectBody: "failed to decode: illegal base64 data at input byte 4\n",
	}, {
		method:     "POST",
		url:        correctURL + tkts[0],
		body:       fmt.Sprintf(`{"pwhash":%q}`, correctHashB64),
		expectCode: http.StatusBadRequest,
		expectBody: "invalid login: name must not be blank\n",
	}, {
		method: "POST",
		url:    correctURL + tkts[0],
		body: fmt.Sprintf(`{"naame":"bob","pwhash":%q}`,
			correctHashB64),
		expectCode: http.StatusBadRequest,
		expectBody: "invalid login: name must not be blank\n",
	}, {
		method:     "POST",
		url:        correctURL,
		body:       correctBody,
		expectCode: http.StatusNotFound,
		expectBody: "404 page not found\n",
	}, {
		method:     "POST",
		url:        correctURL + "some-bad-ticket",
		body:       correctBody,
		expectCode: http.StatusBadRequest,
		expectBody: "uuid: UUID string too short: some-bad-ticket\n",
	}, {
		method:     "POST",
		url:        correctURL + "00000000-0000-0000-0000-000000000000",
		body:       correctBody,
		expectCode: http.StatusNotFound,
		expectBody: "no such ticket `00000000-0000-0000-0000-000000000000`\n",
	}, {
		method: "POST",
		url:    correctURL + tkts[0],
		body: fmt.Sprintf(`{"name":"bodie","pwhash":%q}`,
			correctHashB64),
		expectCode: http.StatusOK,
		expectBody: `{"name":"bodie","coin":0}` + "\n",
		expectUser: `{"name":"bodie","coin":0}`,
	}, {
		method: "POST",
		url:    correctURL + tkts[0],
		body: fmt.Sprintf(`{"name":"bodie","pwhash":%q}`,
			correctHashB64),
		expectCode: http.StatusNotFound,
		expectBody: fmt.Sprintf(
			`no such ticket %#q`+"\n",
			tkts[0],
		),
	}, {
		method: "POST",
		url:    correctURL + tkts[1],
		body: fmt.Sprintf(`{"name":"bodie","pwhash":%q}`,
			correctHashB64),
		expectCode: http.StatusConflict,
		expectBody: "user `bodie` already exists\n",
	}, {
		method: "POST",
		url:    correctURL + tkts[1],
		body: fmt.Sprintf(`{"name":"john","pwhash":%q}`,
			correctHashB64),
		expectCode: http.StatusOK,
		expectBody: `{"name":"john","coin":0}` + "\n",
		expectUser: `{"name":"john","coin":0}`,
	}, {
		method: "POST",
		url:    correctURL + tkts[2],
		body: fmt.Sprintf(`{"name":"bob","coin":5,"pwhash":%q}`,
			correctHashB64),
		expectCode: http.StatusBadRequest,
		expectBody: `invalid login: user cannot be created with coin` + "\n",
	}} {
		c.Logf("test %d:", i)
		c.Logf("  %s %s", test.method, test.url)
		c.Logf("  Body: %#q", test.body)

		r := httprouter.New()
		rest.Incept(r, s.db)
		rdr := bytes.NewBufferString(test.body)
		req := htt.NewRequest(test.method, test.url, rdr)
		w := htt.NewRecorder()
		r.ServeHTTP(w, req)

		c.Check(w.Code, Equals, test.expectCode)
		c.Check(w.Body.String(), Equals, test.expectBody)

		if test.expectUser != "" {
			var b []byte
			u := new(users.User)

			c.Assert(json.Unmarshal([]byte(test.body), u), IsNil)

			c.Assert(s.db.View(func(tx *bolt.Tx) error {
				bs := tx.Bucket(users.UserBucket).Get([]byte(u.Name))
				if bs == nil {
					return &store.MissingError{
						Key:    []byte(u.Name),
						Bucket: users.UserBucket,
					}
				}
				b = bs[:]
				return nil
			}), IsNil)
			c.Check(string(b), Equals, test.expectUser)
		}
	}
}

func (s *RESTSuite) TestInceptHandle(c *C) {

}

package testing

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"net/http"
	htt "net/http/httptest"
	"reflect"

	"github.com/pkg/errors"
	"github.com/synapse-garden/sg-proto/auth"
)

// Bearer returns the appropriate Authorization Header for the given
// Bearer token.
func Bearer(token auth.Token) http.Header {
	return http.Header{"Authorization": []string{
		"Bearer " + base64.StdEncoding.EncodeToString(token),
	}}
}

// WSAuthProtocols returns the websocket Auth Protocol to be passed for
// the given Bearer Token.
func WSAuthProtocols(token auth.Token) []string {
	return []string{
		"Bearer+" + base64.RawURLEncoding.EncodeToString(token),
	}
}

// ExpectResponse makes an HTTP Request on the given Handler, using the
// given URL and REST Method (i.e. PUT, POST, GET, etc.)
func ExpectResponse(
	h http.Handler,
	url, method string,
	bodySend, into, bodyExpect interface{},
	code int,
	header http.Header,
) error {
	var rdr *bytes.Buffer
	if bodySend == nil {
		rdr = new(bytes.Buffer)
	} else if send, err := json.Marshal(bodySend); err != nil {
		return err
	} else {
		rdr = bytes.NewBuffer(send)
	}

	req := htt.NewRequest(method, url, rdr)
	req.Header = header
	w := htt.NewRecorder()
	h.ServeHTTP(w, req)

	if c := w.Code; c != code {
		return errors.Errorf(
			"unexpected response code %d with body %#q",
			c, w.Body,
		)
	}

	bbs := w.Body.Bytes()
	err := json.Unmarshal(bbs, into)
	switch err.(type) {
	case *json.SyntaxError:
		// Some HTTP error code string?
		switch tExp := bodyExpect.(type) {
		case string:
			if tExp != string(bbs) {
				return errors.Errorf("%#q expected, but got response %#q", bodyExpect, bbs)
			}
			return nil
		default:
			return errors.Errorf(
				"unexpected response %#q for expected "+
					"type %T", bbs, bodyExpect,
			)
		}
	case nil:
		if !reflect.DeepEqual(into, bodyExpect) {
			return errors.Errorf(
				"expected response %#v, got response %#v",
				bodyExpect, into,
			)
		}
		return nil
	default:
		return errors.Wrap(err, "failed to unmarshal response body")
	}
}

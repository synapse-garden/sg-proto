package testing

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	htt "net/http/httptest"
	"reflect"

	"github.com/synapse-garden/sg-proto/auth"
	mw "github.com/synapse-garden/sg-proto/rest/middleware"

	"github.com/davecgh/go-spew/spew"
	"github.com/pkg/errors"
)

func makeAuthHeader(kind auth.TokenType, token auth.Token) http.Header {
	return http.Header{string(mw.AuthHeader): []string{
		fmt.Sprintf(
			"%s %s",
			kind, base64.StdEncoding.EncodeToString(token),
		),
	}}
}

// Bearer returns the appropriate Authorization Header for the given
// Bearer token.
func Bearer(token auth.Token) http.Header {
	return makeAuthHeader(auth.BearerType, token)
}

// Admin returns the appropriate Authorization Header for the given
// Admin token.
func Admin(token auth.Token) http.Header {
	return makeAuthHeader(auth.AdminType, token)
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
			"unexpected response code %d (%q) with body %#q",
			c, http.StatusText(c), spew.Sdump(w.Body),
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
				return errors.Errorf(
					"%#q expected, but got response %#q",
					tExp, bbs,
				)
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
				"expected response %s, got response %s",
				spew.Sdump(bodyExpect),
				spew.Sdump(into),
			)
		}
		return nil
	default:
		return errors.Wrap(err, "failed to unmarshal response body")
	}
}

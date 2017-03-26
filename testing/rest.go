package testing

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	htt "net/http/httptest"
	"reflect"
	"strings"

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

// Options returns the appropriate Allow header for the given Options.
func Options(methods ...string) http.Header {
	return http.Header{"Allow": []string{strings.Join(methods, ", ")}}
}

// WSAuthProtocols returns the websocket Auth Protocol to be passed for
// the given Bearer Token.
func WSAuthProtocols(token auth.Token) []string {
	return []string{
		"Bearer+" + base64.RawURLEncoding.EncodeToString(token),
	}
}

var OKHeader = http.Header{
	"Content-Type": []string{"text/plain; charset=utf-8"},
}

var FailHeader = http.Header{
	"Content-Type":           {"text/plain; charset=utf-8"},
	"X-Content-Type-Options": {"nosniff"},
}

// ExpectResponse makes an HTTP Request on the given Handler, using the
// given URL and REST Method (i.e. PUT, POST, GET, etc.)
//
// Multiple expected headers may be added.  Each will be added into one
// map of expected headers using http.Header.Add; if the same header key
// is passed twice, the last one given will be used.
//
// If no expected headers are passed, defaults will be selected based on
// expected HTTP status code: if 200,
func ExpectResponse(
	h http.Handler,
	url, method string,
	bodySend, into, bodyExpect interface{},
	code int,
	header http.Header,
	expectHeaders ...http.Header,
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

	switch {
	case expectHeaders != nil:
		// Just use the ones passed.
	case code == http.StatusOK && method == "DELETE":
		// DELETE doesn't get headers.
		expectHeaders = []http.Header{}
	case code == http.StatusOK:
		// Responses with bodies use these headers.
		expectHeaders = []http.Header{OKHeader}
	default:
		// Responses with error messages use these headers.
		expectHeaders = []http.Header{FailHeader}
	}

	expectHeader := make(http.Header)
	for _, hdr := range expectHeaders {
		for k, vs := range hdr {
			for _, v := range vs {
				expectHeader.Add(k, v)
			}
		}
	}

	wHdr := w.Header()
	for k, vs := range expectHeader {
		if err := compareHdrValues(k, wHdr[k], vs); err != nil {
			return errors.Wrapf(err,
				"headers did not match:\n"+
					"got %s\n"+
					"expected %s",
				spew.Sdump(wHdr),
				spew.Sdump(expectHeader),
			)
		}
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
	default:
		return errors.Wrap(err, "failed to unmarshal response body")
	}

	return nil
}

func compareHdrValues(k string, expect, got []string) error {
	if len(expect) != len(got) {
		return errors.Errorf(`"Allow" header %+v did not match `+
			"expected %+v", expect, got)
	}

	if k != "Allow" {
		// "ALLOW" header is a special case.  In general, we
		// just expect the values to match.
		if !reflect.DeepEqual(expect, got) {
			return errors.Errorf("headers did not match:\n"+
				"got %s\n"+
				"expected: %s",
				spew.Sdump(got),
				spew.Sdump(expect),
			)
		}

		return nil
	}

	// Expect only one string slice.
	if le := len(expect); le != 1 {
		return errors.Errorf("should expect only one "+
			`"ALLOW" header; expected %d`, le)
	}

	expV := make(map[string]bool)
	for _, v := range strings.Split(expect[0], ", ") {
		expV[v] = true
	}

	// Must have same contents, order unimportant.
	for _, v := range strings.Split(got[0], ", ") {
		if found := expV[v]; !found {
			// Unexpected value.
			return errors.Errorf("found unexpected"+
				`"Allow" header value %q`, v)
		}

		delete(expV, v)
	}

	// Some expected OPTION verbs remained unseen.
	if lRemain := len(expV); lRemain > 0 {
		var rpt []string
		for v := range expV {
			rpt = append(rpt, v)
		}

		return errors.Errorf("%d OPTION verbs were "+
			"not seen: %+v", lRemain, rpt)
	}

	return nil
}

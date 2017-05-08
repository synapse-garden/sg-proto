package client

import (
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/synapse-garden/sg-proto/auth"

	"github.com/pkg/errors"
)

var customClient = &http.Client{}

// SetCustomCert sets the internal HTTP Client's TLS config to accept
// the passed certificate bytes.  It is not safe to do this concurrently
// with HTTP requests in this package.  Any error will reset the
// internal Client back to its defaults.
func SetCustomCert(cert []byte) error {
	roots := x509.NewCertPool()
	if !roots.AppendCertsFromPEM(cert) {
		return errors.New("failed to append cert")
	}

	customClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{RootCAs: roots},
	}

	return nil
}

// DecodeDelete makes an HTTP DELETE request to the given resource under
// the given RequestTransforms.
func DecodeDelete(resource string, xfs ...RequestTransform) error {
	req, err := http.NewRequest(http.MethodDelete, resource, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create HTTP request")
	}

	if res, err := customClient.Do(transform(req, xfs...)); err != nil {
		return errors.Wrap(err, "failed to make HTTP request")
	} else if stat := res.StatusCode; stat != http.StatusOK {
		err := errors.Errorf("HTTP request failed with status %d (%s)",
			stat, http.StatusText(stat),
		)
		defer res.Body.Close()
		bs, e := ioutil.ReadAll(res.Body)
		if e != nil {
			return errors.Wrapf(err, "failed to read error body after bad request: %s", e.Error())
		}

		return errors.Wrap(err, fmt.Sprintf("%#q", bs))
	} else {
		return res.Body.Close()
	}
}

// DecodePost makes an HTTP POST to the given resource using a JSON
// marshaled request body from 'body', and applying xfs to the request.
func DecodePost(
	v interface{},
	resource string,
	body interface{},
	xfs ...RequestTransform,
) error {
	var bs []byte
	if body != nil {
		var err error
		bs, err = json.Marshal(body)
		if err != nil {
			return errors.Wrap(err, "failed to marshal JSON body")
		}
	}
	req, err := http.NewRequest(http.MethodPost, resource, bytes.NewBuffer(bs))
	if err != nil {
		return errors.Wrap(err, "failed to create HTTP request")
	}

	if res, err := customClient.Do(transform(req, xfs...)); err != nil {
		return errors.Wrap(err, "failed to make HTTP request")
	} else if stat := res.StatusCode; stat != http.StatusOK {
		err := errors.Errorf("HTTP request failed with status %d (%s)",
			stat, http.StatusText(stat),
		)
		defer res.Body.Close()
		bs, e := ioutil.ReadAll(res.Body)
		if e != nil {
			return errors.Wrapf(err, "failed to read error body after bad request: %s", e.Error())
		}

		return errors.Wrap(err, fmt.Sprintf("%#q", bs))
	} else {
		defer res.Body.Close()
		return json.NewDecoder(res.Body).Decode(v)
	}
}

// DecodeGet unmarshals the given resource (after applying xfs) into v
// using an HTTP GET.
func DecodeGet(
	v interface{},
	resource string,
	xfs ...RequestTransform,
) error {
	req, err := http.NewRequest(http.MethodGet, resource, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create HTTP request")
	}
	if res, err := customClient.Do(transform(req, xfs...)); err != nil {
		return errors.Wrap(err, "failed to make HTTP request")
	} else if stat := res.StatusCode; stat != http.StatusOK {
		err := errors.Errorf("HTTP request failed with status %d (%s)",
			stat, http.StatusText(stat),
		)
		defer res.Body.Close()
		bs, e := ioutil.ReadAll(res.Body)
		if e != nil {
			return errors.Wrapf(err, "failed to read error body after bad request: %s", e.Error())
		}

		return errors.Wrap(err, fmt.Sprintf("%#q", bs))
	} else {
		defer res.Body.Close()
		return json.NewDecoder(res.Body).Decode(v)
	}
}

// Delete makes an HTTP DELETE request on the given resource using xfs.
func Delete(resource string, xfs ...RequestTransform) error {
	req, err := http.NewRequest(http.MethodDelete, resource, nil)
	if err != nil {
		return errors.Wrap(err, "failed to create HTTP request")
	}
	if res, err := customClient.Do(transform(req, xfs...)); err != nil {
		return errors.Wrap(err, "failed to make HTTP request")
	} else if stat := res.StatusCode; stat != http.StatusOK {
		err := errors.Errorf("HTTP request failed with status %d (%s)",
			stat, http.StatusText(stat),
		)
		defer res.Body.Close()
		bs, e := ioutil.ReadAll(res.Body)
		if e != nil {
			return errors.Wrapf(err, "failed to read error body after bad request: %s", e.Error())
		}

		return errors.Wrap(err, fmt.Sprintf("%#q", bs))
	} else {
		return res.Body.Close()
	}
}

// Param is a URL param builder.  Use ApplyParams to get the parameterized
// URL.
type Param fmt.Stringer

// Filter is a Param for filtering by keyword.
type Filter string

// String implements Param on Filter.
func (f Filter) String() string {
	return "filter=" + string(f)
}

// ApplyParams applies the given Params to generate a sequence of URL parameters.
func ApplyParams(ps ...Param) string {
	if len(ps) == 0 {
		return ""
	}
	str := bytes.NewBufferString("?")
	for _, p := range ps {
		str.WriteString("&" + p.String())
	}
	return str.String()
}

// A RequestTransform can be applied to transform an *http.Request.
type RequestTransform func(*http.Request) *http.Request

func transform(req *http.Request, xfs ...RequestTransform) *http.Request {
	for _, xf := range xfs {
		req = xf(req)
	}
	return req
}

// AuthHeader adds an Authorization header of the given TokenType
// (auth.BearerType or auth.RefreshType.)
func AuthHeader(t auth.TokenType, tk auth.Token) RequestTransform {
	return func(req *http.Request) *http.Request {
		req.Header.Add("Authorization", t.String()+" "+tk.String())
		return req
	}
}

// AdminHeader adds an Authorization header of type Admin using the
// given API key.
func AdminHeader(apiKey string) RequestTransform {
	return func(req *http.Request) *http.Request {
		req.Header.Add("Authorization", "Admin "+apiKey)
		return req
	}
}

package client

import (
	"crypto/sha256"
	"crypto/tls"
	"errors"
	"fmt"
	"net/http"
	"net/url"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/stream"
	"github.com/synapse-garden/sg-proto/users"

	ws "golang.org/x/net/websocket"
)

// Client is a client for an SG backend.
type Client struct {
	State   State
	APIKey  string
	Backend *url.URL
}

// Info sets the given info based on the backend's /source.
func (c *Client) Info(i *rest.SourceInfo) error {
	return DecodeGet(i, c.Backend.String()+"/source")
}

// CreateTickets creates incept tickets using the given API key.
func (c *Client) CreateTickets(into *[]incept.Ticket, n int) error {
	if c.APIKey == "" {
		return errors.New("client must have a valid admin API key")
	}

	str := c.Backend.String() + "/admin/tickets"
	if n > 1 {
		str = fmt.Sprintf("%s?count=%d", str, n)
	}
	return DecodePost(into, str, nil, AdminHeader(c.APIKey))
}

// CreateLogin creates a user with the given name and password (which is
// hashed before sending) and unmarshals the response into the given *Login.
func (c *Client) CreateLogin(
	l *auth.Login,
	ticket, name, pw string,
) error {
	b := sha256.Sum256([]byte(pw))
	return DecodePost(l,
		c.Backend.String()+"/incept/"+ticket,
		&auth.Login{
			User:   users.User{Name: name},
			PWHash: b[:],
		},
	)
}

// VerifyAdmin checks that the Client's APIKey is a valid Admin key.
func (c *Client) VerifyAdmin(key string) error {
	var ok bool
	return DecodeGet(&ok, c.Backend.String()+"/admin/verify",
		AdminHeader(key))
}

// Login uses the given *auth.Login's Token to get a Session.
func (c *Client) Login(l *auth.Login) error {
	s := c.State.Session
	return DecodePost(s, c.Backend.String()+"/tokens", l)
}

// Logout deletes the given *auth.Login's Token.
func (c *Client) Logout() error {
	if sesh := c.State.Session; sesh == nil {
		return errors.New("nil session")
	} else if t := sesh.Token; t == nil {
		return errors.New("nil session token")
	} else {
		return Delete(c.Backend.String()+"/tokens",
			AuthHeader(auth.BearerType, t))
	}
}

// GetProfile gets the User for the given Session.
func (c *Client) GetProfile(u *users.User) error {
	s := c.State.Session
	return DecodeGet(u, c.Backend.String()+"/profile",
		AuthHeader(auth.BearerType, s.Token))
}

// DeleteProfile deletes the Session owner's profile.
func (c *Client) DeleteProfile() error {
	s := c.State.Session
	return Delete(c.Backend.String()+"/profile",
		AuthHeader(auth.BearerType, s.Token))
}

// CreateStream creates a new Stream belonging to the Session owner.
func (c *Client) CreateStream(str *stream.Stream) error {
	if sesh := c.State.Session; sesh == nil {
		return errors.New("nil session")
	} else if t := sesh.Token; t == nil {
		return errors.New("nil session token")
	} else {
		return DecodePost(str,
			c.Backend.String()+"/streams",
			str,
			AuthHeader(auth.BearerType, t),
		)
	}
}

// GetStream gets a Stream by ID.
func (c *Client) GetStream(str *stream.Stream, id string) error {
	if sesh := c.State.Session; sesh == nil {
		return errors.New("nil session")
	} else if t := sesh.Token; t == nil {
		return errors.New("nil session token")
	} else {
		return DecodeGet(str,
			c.Backend.String()+"/streams/"+id,
			AuthHeader(auth.BearerType, t),
		)
	}
}

// AllStreams gets the User's owned Streams.
func (c *Client) AllStreams(strs *[]*stream.Stream, filters ...Param) error {
	if sesh := c.State.Session; sesh == nil {
		return errors.New("nil session")
	} else if t := sesh.Token; t == nil {
		return errors.New("nil session token")
	} else {
		return DecodeGet(strs, fmt.Sprintf(
			"%s/streams%s",
			c.Backend,
			ApplyParams(filters...)),
			AuthHeader(auth.BearerType, t),
		)
	}
}

// GetStreamWS opens and returns a *golang.org/x/net/websocket.Conn.
func (c *Client) GetStreamWS(id string) (*ws.Conn, error) {
	s := c.State.Session
	if c.State.Session == nil {
		return nil, fmt.Errorf("cannot get stream with a nil Session")
	}
	backend := *c.Backend
	switch backend.Scheme {
	case "http":
		backend.Scheme = "ws"
	case "https":
		backend.Scheme = "wss"
	}

	backend.Path += "/streams/" + id + "/start"
	var conf *tls.Config
	if t, ok := customClient.Transport.(*http.Transport); ok {
		conf = t.TLSClientConfig
	}
	return ws.DialConfig(&ws.Config{
		Location:  &backend,
		Origin:    &url.URL{},
		TlsConfig: conf,
		Version:   ws.ProtocolVersionHybi13,
		Header: http.Header{
			"Authorization": {"Bearer " + s.Token.String()},
		},
	})
}

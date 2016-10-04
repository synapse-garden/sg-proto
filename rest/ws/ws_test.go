package ws_test

import (
	"io"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/synapse-garden/sg-proto/rest/ws"

	xws "golang.org/x/net/websocket"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type WSSuite struct{}

var _ = Suite(&WSSuite{})

func (s *WSSuite) TestCheck(c *C) {
	c.Assert(ws.Check(nil, nil), IsNil)
}

type sr struct {
	*io.PipeReader
	*io.PipeWriter
}

func (s sr) Send(bs []byte) error {
	_, err := s.Write(bs)
	return err
}

func (s sr) Recv() ([]byte, error) {
	bs := make([]byte, 1024)
	n, err := s.Read(bs)
	if n > 0 {
		return bs[0:n], err
	}
	return nil, err
}

func (s *WSSuite) TestBind(c *C) {
	r, w := io.Pipe()
	srv := sr{r, w}

	server := httptest.NewServer(xws.Server{
		Handshake: ws.Check,
		Handler:   ws.Bind(srv, ws.DefaultRead),
	})

	u, err := url.Parse(server.URL)
	c.Assert(err, IsNil)
	u.Scheme = "ws"

	conn, err := xws.DialConfig(&xws.Config{
		Location: u,
		Origin:   &url.URL{},
		Version:  xws.ProtocolVersionHybi13,
	})
	c.Assert(err, IsNil)

	_, err = conn.Write([]byte("hello"))
	c.Assert(err, IsNil)

	bs := make([]byte, 1024)
	n, err := conn.Read(bs)
	c.Assert(err, IsNil)
	c.Check(string(bs[0:n]), Equals, "hello")

	_, err = conn.Write([]byte(strings.Repeat("!", 128)))
	c.Assert(err, IsNil)

	bs = make([]byte, 1024)
	n, err = conn.Read(bs)
	c.Assert(err, IsNil)
	c.Check(string(bs[0:n]), Equals, strings.Repeat("!", 128))

	c.Assert(srv.PipeReader.Close(), IsNil)
	c.Assert(srv.PipeWriter.Close(), IsNil)

	c.Assert(conn.Close(), IsNil)
}

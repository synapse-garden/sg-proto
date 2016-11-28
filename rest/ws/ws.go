package ws

import (
	"io"
	"log"
	"net/http"

	xws "golang.org/x/net/websocket"
)

// DefaultBuffer is the size of the DefaultRead buffer.
const DefaultBuffer = 64

// Handshaker is a function that checks an HTTP request for websockets.
type Handshaker func(config *xws.Config, req *http.Request) error

// Check is a Handshaker which does nothing.
func Check(config *xws.Config, req *http.Request) error {
	return nil
}

var _ = Handshaker(Check)

type Binder interface {
	Bind(*xws.Conn)
}

type Sender interface {
	Send([]byte) error
}

type Recver interface {
	Recv() ([]byte, error)
}

type SendRecver interface {
	Sender
	Recver
}

type RecvCloser interface {
	io.Closer
	Recver
}

// SocketReader is a function that gets []byte from a Websocket conn.
// If there's a formatting error which the client needs to be told
// about, the bool value will be false.
type SocketReader func(c *xws.Conn) ([]byte, bool, error)

// DefaultRead receives raw bytes from the websocket.  It starts with a
// buffer of 128 and grows it as needed.  It never rejects a message
// as malformed.
func DefaultRead(c *xws.Conn) ([]byte, bool, error) {
	buf := make([]byte, DefaultBuffer)
	n := 0

	for {
		max := cap(buf)
		m, err := c.Read(buf[n:max])
		n += m
		if n == max {
			// Buffer full, double the size and try again.
			buf = append(buf, make([]byte, cap(buf)*2)...)
			continue
		}
		return buf[0:n], true, err
	}
}

// Bind routes messages and errors between a SendRecver and a websocket
// connection.
func Bind(sr SendRecver, read SocketReader) xws.Handler {
	if read == nil {
		read = DefaultRead
	}
	return func(c *xws.Conn) {
		ch := chans{
			Conn:       c,
			SendRecver: sr,

			errs: make(chan error, 10),
			done: make(chan struct{}),
			fail: make(chan struct{}),
		}

		go ch.WriteErrors()
		go ch.RecvWrite()
		go ch.ReadSend(read)

		<-ch.done
		c.Close()
	}
}

// BindRead receives messages from a Recver and writes them to the Conn.
// It should be used in place of Bind for one-way messaging from the
// backend to the websocket client.
func BindRead(r Recver) xws.Handler {
	return func(c *xws.Conn) {
		// Recv from r; pass result to c Write if no error.
		for bs, err := r.Recv(); err == nil; bs, err = r.Recv() {
			if _, err = c.Write(bs); err != nil {
				break
			}
		}

		if err := c.Close(); err != nil {
			log.Printf("failed to close Websocket: %s", err.Error())
		}
	}
}

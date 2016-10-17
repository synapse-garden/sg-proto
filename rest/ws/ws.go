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
//
// TODO: Tighten this up.  A lot.
func Bind(sr SendRecver, read SocketReader) xws.Handler {
	if read == nil {
		read = DefaultRead
	}
	return func(c *xws.Conn) {
		done, fail := make(chan struct{}), make(chan struct{})
		errs := make(chan error, 10)

		go func() {
			defer close(fail)
			// Receive from sr; pass result to c Write.
			for {
				select {
				case err := <-errs:
					_, err = c.Write([]byte(err.Error()))
					if err != nil {
						return
					}
				case <-done:
					return
				default:
				}

				if bs, err := sr.Recv(); err != nil {
					return
				} else if _, err = c.Write(bs); err != nil {
					return
				}
			}
		}()

		go func() {
			defer close(done)
			// Receive from websocket; pass result to sr Send.
			for {
				select {
				case <-fail:
					return
				default:
				}
				if bs, ok, err := read(c); err == io.EOF {
					// Websocket was closed.
					return
				} else if err != nil {
					log.Printf("failed to read from socket: %s", err.Error())
					return
				} else if !ok {
					// Formatting error.  Tell the
					// frontend, then move on.
					errs <- err
				} else if err = sr.Send(bs); err != nil {
					log.Printf("failed to send to Sender: %s", err.Error())
					return
				}
			}
		}()

		<-done

		if err := c.Close(); err != nil {
			log.Printf("failed to close Websocket: %s", err.Error())
		}
	}
}

// BindRead receives messages from the websocket SendRecver and a
// websocket connection.
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

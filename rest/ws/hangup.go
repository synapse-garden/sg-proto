package ws

import (
	"io"
	"log"

	"github.com/synapse-garden/sg-proto/stream/river"

	"github.com/pkg/errors"
	xws "golang.org/x/net/websocket"
)

// HangupRecver is a Responder river which closes its signal channel
// after Sending.
type HangupRecver struct {
	river.Responder
	sr     SocketReader
	signal chan struct{}
}

// MakeHangup creates a new HangupRecver from the given river.Responder
// and SocketReader.  If the SocketReader is nil, DefaultReader will be
// used.
func MakeHangup(r river.Responder, sr SocketReader) HangupRecver {
	if sr == nil {
		sr = DefaultRead
	}
	return HangupRecver{
		Responder: r,
		sr:        sr,
		signal:    make(chan struct{}),
	}
}

// Send implements river.Responder.Send on HangupRecver, closing its
// signal channel and calling Send on the underlying Responder.  Make
// sure this is only called once.
func (h HangupRecver) Send(msg []byte) error {
	close(h.signal)
	return h.Responder.Send(msg)
}

// Read is a SocketReader which returns io.EOF when the HangupRecver
// has Sent and closes the Conn.
//
// TODO: Tighten this up a lot!
func (h HangupRecver) Read(c *xws.Conn) (bs []byte, ok bool, e error) {
	done := make(chan struct{})
	select {
	case <-h.signal:
		return nil, true, errors.Wrap(io.EOF, "stream hung up")
	default:
		go func() {
			select {
			case <-h.signal:
				if _, err := c.Write([]byte(errors.Wrap(
					io.EOF, "stream hung up",
				).Error())); err != nil {
					log.Printf("failed to write "+
						"hangup message: %s",
						err.Error())
				}
				if err := c.Close(); err != nil {
					log.Printf("failed to close "+
						"websocket after hangup "+
						"request: %s", err.Error())
				}
			case <-done:
			}
		}()
		defer func() {
			close(done)
			select {
			case <-h.signal:
				bs, ok, e = nil, true, errors.Wrap(io.EOF, "stream hung up")
			default:
			}
		}()
		return h.sr(c)
	}
}

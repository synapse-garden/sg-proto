package ws

import (
	"io"
	"log"

	"github.com/synapse-garden/sg-proto/stream/river"

	"github.com/pkg/errors"
	xws "golang.org/x/net/websocket"
)

// HangupSender is a Responder river which closes its signal channel
// after Sending.
type HangupSender struct {
	river.Responder
	sr     SocketReader
	signal chan struct{}
}

// MakeHangup creates a new HangupSender from the given river.Responder
// and SocketReader.  If the SocketReader is nil, DefaultReader will be
// used.
func MakeHangup(r river.Responder, sr SocketReader) HangupSender {
	if sr == nil {
		sr = DefaultRead
	}
	return HangupSender{
		Responder: r,
		sr:        sr,
		signal:    make(chan struct{}),
	}
}

// Send implements river.Responder.Send on HangupSender, closing its
// signal channel and calling Send on the underlying Responder.  Make
// sure this is only called once.
func (h HangupSender) Send(msg []byte) error {
	close(h.signal)
	return h.Responder.Send(msg)
}

// Read is a SocketReader which returns io.EOF when the HangupSender
// has Sent and closes the Conn.
//
// TODO: Tighten this up a lot!
func (h HangupSender) Read(c *xws.Conn) (bs []byte, ok bool, e error) {
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

// HangupRecver is a type which is hung up when it sends its response,
// and upon being hung up, will close the River socket used in its Recv.
type HangupRecver struct {
	river.Responder

	socket RecvCloser
	signal chan struct{}
}

// hangupRecver wraps the given Recver with a closable signal channel
// for hangup.
type hangupRecver struct {
	RecvCloser
	signal chan struct{}
}

// Recv implements Recver on hangupRecver.  If its signal channel is
// closed, it will close the underlying RecvCloser.
func (h hangupRecver) Recv() ([]byte, error) {
	done := make(chan struct{})
	select {
	case <-h.signal:
		return nil, errors.Wrap(io.EOF, "stream hung up")
	default:
	}
	go func() {
		select {
		case <-h.signal:
			h.Close()
		case <-done:
		}
	}()
	defer close(done)
	return h.RecvCloser.Recv()
}

// MakeHangupRecver makes a new HangupRecver which will close its Recver
// socket when it responds to a hangup Survey.
func MakeHangupRecver(rsp river.Responder, r RecvCloser) HangupRecver {
	signal := make(chan struct{})
	return HangupRecver{
		Responder: rsp,
		socket: hangupRecver{
			RecvCloser: r,
			signal:     signal,
		},
		signal: signal,
	}
}

// Send implements Responder.Send on HangupRecver, closing its signal
// channel and calling Send on the underlying Responder.  Make sure this
// is only called once!
func (h HangupRecver) Send(bs []byte) error {
	close(h.signal)
	return h.Responder.Send(bs)
}

// Recver returns the HangupRecver's Recv socket, which is closed when
// the HangupRecver replies to a hangup survey.
func (h HangupRecver) Recver() Recver {
	return h.socket
}

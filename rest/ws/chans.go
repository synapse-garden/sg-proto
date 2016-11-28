package ws

import (
	"encoding/json"
	"io"
	"log"

	"github.com/pkg/errors"

	xws "golang.org/x/net/websocket"
)

// MessageError is a JSON-serializable error message.
type MessageError struct{ Error error }

// MarshalJSON implements JSON.Marshaler on MessageError.
func (m MessageError) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		Error string `json:"error"`
	}{m.Error.Error()})
}

// chans is a simple struct for binding a set of done, errors, and fail
// channels from a typical websocket bind to simplify the ws.Bind func.
type chans struct {
	*xws.Conn
	SendRecver

	errs       chan error
	done, fail chan struct{}
}

// WriteErrors receives errors from the chans error channel, and writes
// them to the Websocket.
func (ch chans) WriteErrors() {
	for {
		select {
		case err := <-ch.errs:
			bytes, err := json.Marshal(MessageError{err})
			if err != nil {
				log.Fatalf("failed to marshal MessageError: %#v", err)
			} else if _, err = ch.Write(bytes); err != nil {
				log.Printf("failed to write websocket "+
					"error to reader: %s",
					err.Error())
				return
			}
		case <-ch.done:
			return
		}
	}
}

// RecvWrite receives messages from the SendRecver and writes them to
// the Conn.  If an error is received from errs, it will write it to the
// Conn using err.Error().  When done is closed, it will return.  Any
// error on Recv or Write will cause it to return silently.
func (ch chans) RecvWrite() {
	defer close(ch.fail)
	// Receive from sr; pass result to c Write.
	for {
		if bs, err := ch.Recv(); err != nil {
			return
		} else if _, err = ch.Write(bs); err != nil {
			return
		}
	}
}

// ReadSend uses the given SocketReader to read bytes from the Conn.  In
// case of errors.Cause(err) == io.EOF, it will return silently.
// Otherwise, it will Send the read bytes on its SendRecver.
func (ch chans) ReadSend(read SocketReader) {
	defer close(ch.done)
	// Receive from websocket; pass result to sr Send.
	for {
		select {
		case <-ch.fail:
			return
		default:
		}
		if bs, ok, err := read(ch.Conn); errors.Cause(err) == io.EOF {
			return
		} else if ok && err != nil {
			// Error, but not a parse error.
			log.Printf("failed to read from socket: %s", err.Error())
			return
		} else if !ok && err != nil {
			// Content error.  Tell the frontend, then move on.
			ch.errs <- errors.Wrap(err, "malformed message")
		} else if !ok {
			// Content error, but no specifics.  Tell the
			// frontend, then move on.
			ch.errs <- errors.Errorf("malformed message: %#q", bs)
		} else if err = ch.Send(bs); err != nil {
			// Not bad content, and no parse error, but
			// failed to send.
			log.Printf("failed to send to Sender: %s", err.Error())
			return
		}
	}
}

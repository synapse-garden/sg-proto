package convo

import (
	"encoding/json"

	"github.com/synapse-garden/sg-proto/stream"
	"github.com/synapse-garden/sg-proto/util"

	"github.com/pkg/errors"
	ws "golang.org/x/net/websocket"
)

const ConvoMessage int = 1

// Sender is a User who reads stream.Messages from a websocket Conn and
// binds their contents into a new convo.Message with the attached user.
type Sender struct {
	util.Timer
	Name string
}

// Read is a SocketReader for passing to ws.Bind which wraps a received
// stream.Message contents with the Sender's userID in a convo.Message,
// and marshals the convo.Message into bytes.
//
// A false bool return value indicates a normal or non-fatal error, such
// as a syntax error, so that the caller will know to notify the sender
// and continue to the next Read.
func (s Sender) Read(conn *ws.Conn) ([]byte, bool, error) {
	msg := new(stream.Message)
	if err := ws.JSON.Receive(conn, msg); err != nil {
		switch err.(type) {
		// Just a syntax error, try again with a correct value.
		case *json.SyntaxError:
			return nil, false, errors.Wrap(err,
				"failed to unmarshal stream.Message")
		case *json.UnmarshalTypeError:
			return nil, false, errors.Wrap(err,
				"failed to unmarshal stream.Message")
		case *json.UnsupportedValueError:
			return nil, false, errors.Wrap(err,
				"failed to unmarshal stream.Message")
		default:
			// Something else went wrong.
			return nil, true, errors.Wrap(err,
				"failed to unmarshal stream.Message")
		}
	}

	bs, err := json.Marshal(&Message{
		Content:   msg.Content,
		Sender:    s.Name,
		Timestamp: s.Now(),
	})

	if err != nil {
		return nil, false, err
	}

	return bs, true, nil
}

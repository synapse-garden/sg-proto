package convo_test

import (
	"encoding/json"
	"net/http"
	htt "net/http/httptest"
	"net/url"
	"reflect"
	"time"

	"github.com/synapse-garden/sg-proto/convo"
	"github.com/synapse-garden/sg-proto/stream"
	"github.com/synapse-garden/sg-proto/testing"

	"github.com/julienschmidt/httprouter"
	"github.com/pkg/errors"
	ws "golang.org/x/net/websocket"
	. "gopkg.in/check.v1"
)

type ch struct {
	handler func(*ws.Conn) ([]byte, bool, error)

	bytes    chan []byte
	bools    chan bool
	errs     chan error
	die      <-chan struct{}
	finished chan struct{}
}

func (c ch) handle(conn *ws.Conn) {
	defer close(c.finished)
	for {
		select {
		case <-c.die:
			close(c.finished)
			return
		default:
		}
		bs, ok, err := c.handler(conn)
		c.bytes <- bs
		c.bools <- ok
		c.errs <- err
	}
}

func setupWSHandler(c *C,
	r *httprouter.Router,
	handler ws.Handler,
) *htt.Server {
	r.Handler("GET", "/", ws.Server{
		Handshake: func(*ws.Config, *http.Request) error { return nil },
		Handler:   handler,
	})
	// Make a testing server to run it.
	return htt.NewServer(r)
}

func dialWS(c *C, path string) *ws.Conn {
	urlLoc, err := url.Parse(path)
	c.Assert(err, IsNil)

	urlLoc.Scheme = "ws"
	conn, err := ws.DialConfig(&ws.Config{
		Location: urlLoc,
		Origin:   &url.URL{},
		Version:  ws.ProtocolVersionHybi13,
	})
	c.Assert(err, IsNil)

	return conn
}

func (s *ConvoSuite) TestSenderRead(c *C) {
	var (
		now = time.Now()

		sender = convo.Sender{
			Timer: testing.Timer(now),
			Name:  "bob",
		}

		bytes    = make(chan []byte)
		bools    = make(chan bool)
		errs     = make(chan error)
		die      = make(chan struct{})
		finished = make(chan struct{})

		chans = ch{
			handler: sender.Read,

			bytes:    bytes,
			bools:    bools,
			errs:     errs,
			die:      die,
			finished: finished,
		}

		srv = setupWSHandler(c, httprouter.New(), chans.handle)
	)
	defer srv.Close()
	defer close(die)

	conn := dialWS(c, srv.URL)

	_, err := conn.Write([]byte("hello"))
	c.Assert(err, IsNil)

	c.Check(<-chans.bytes, IsNil)
	c.Check(<-chans.bools, Equals, false)
	err = <-chans.errs
	c.Check(err, ErrorMatches, `.* invalid character.*`)
	c.Check(reflect.TypeOf(errors.Cause(err)),
		DeepEquals,
		reflect.TypeOf(&json.SyntaxError{}),
	)

	select {
	case err := <-chans.errs:
		c.Errorf("received unexpected second error: %s", err.Error())
	case <-chans.finished:
		c.Error("websocket unexpectedly closed")
	default:
	}

	okBytes, err := json.Marshal(stream.Message{Content: "hello"})
	c.Assert(err, IsNil)
	expectBytes, err := json.Marshal(&convo.Message{
		Content:   "hello",
		Sender:    sender.Name,
		Timestamp: now,
	})

	_, err = conn.Write(okBytes)
	c.Assert(err, IsNil)

	c.Check(<-chans.bytes, DeepEquals, expectBytes)
	c.Check(<-chans.bools, Equals, true)
	c.Check(<-chans.errs, IsNil)

	select {
	case err := <-chans.errs:
		c.Errorf("received unexpected second error: %s", err.Error())
	case <-chans.finished:
		c.Error("websocket unexpectedly closed")
	default:
	}
}

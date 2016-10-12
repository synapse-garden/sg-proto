package client_test

import (
	"bytes"
	"encoding/base64"
	htt "net/http/httptest"
	"net/url"
	"os"
	"testing"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/client"
	"github.com/synapse-garden/sg-proto/rest"
	sgt "github.com/synapse-garden/sg-proto/testing"

	"github.com/boltdb/bolt"
	uuid "github.com/satori/go.uuid"
	. "gopkg.in/check.v1"
)

func Test(t *testing.T) { TestingT(t) }

type ClientSuite struct {
	db     *bolt.DB
	tmpDir string

	srv *htt.Server
	cli *client.Client

	rbuf, wbuf *bytes.Buffer
}

var _ = Suite(&ClientSuite{})

var src = &rest.SourceInfo{
	License:    "Affero GPL V3",
	LicensedTo: "SynapseGarden 2016",
	Location:   "https://github.com/synapse-garden/sg-proto",
}

func (s *ClientSuite) SetUpTest(c *C) {
	db, tmpDir, err := sgt.TempDB("sg-test")
	c.Assert(err, IsNil)
	key := uuid.NewV4()
	r, err := rest.Bind(db, src, auth.Token(key.Bytes()))
	c.Assert(err, IsNil)
	srv := htt.NewServer(r)
	u, err := url.Parse(srv.URL)
	c.Assert(err, IsNil)
	rbuf, wbuf := new(bytes.Buffer), new(bytes.Buffer)
	cli := &client.Client{
		Backend: u,
		APIKey:  base64.StdEncoding.EncodeToString(key.Bytes()),
		State:   client.MakeState(rbuf, wbuf),
	}

	s.tmpDir = tmpDir
	s.db, s.srv, s.cli = db, srv, cli
}

func (s *ClientSuite) TearDownTest(c *C) {
	if db := s.db; db != nil {
		c.Assert(sgt.CleanupDB(db), IsNil)
		c.Assert(os.Remove(s.tmpDir), IsNil)
	}

	if srv := s.srv; srv != nil {
		srv.Close()
	}
}

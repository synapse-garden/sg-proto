package client_test

import (
	"crypto/sha256"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/users"

	. "gopkg.in/check.v1"
)

func (s *ClientSuite) TestInfo(c *C) {
	info := new(rest.SourceInfo)
	c.Assert(s.cli.Info(info), IsNil)
	c.Check(info, DeepEquals, src)
}

func (s *ClientSuite) TestCreateTickets(c *C) {
	ak := s.cli.APIKey
	s.cli.APIKey = ""

	var ts []incept.Ticket
	c.Check(s.cli.CreateTickets(&ts, 5), ErrorMatches, `client must have a valid admin API key`)
	c.Check(ts, IsNil)

	s.cli.APIKey = ak
	c.Assert(s.cli.CreateTickets(&ts, 5), IsNil)
	c.Check(len(ts), Equals, 5)
	for _, t := range ts {
		c.Check(len(t), Equals, 16)
		uu, err := uuid.FromBytes(t.Bytes())
		c.Check(err, IsNil)
		c.Check(uuid.Equal(uu, uuid.UUID(t)), Equals, true)
		// It's a valid UUID
	}
}

func (s *ClientSuite) TestCreateLogin(c *C) {
	var ts []incept.Ticket
	c.Assert(s.cli.CreateTickets(&ts, 1), IsNil)
	c.Check(len(ts), Equals, 1)

	l := new(auth.Login)
	ticket := ts[0].String()
	c.Assert(s.cli.CreateLogin(l, ticket, "bodie", "hello"), IsNil)
	c.Check(l, DeepEquals, &auth.Login{
		User: users.User{
			Name: "bodie",
			Coin: 0,
		},
	})

	// Does it work?
	bs := sha256.Sum256([]byte("hello"))
	c.Assert(s.cli.Login(&auth.Login{
		User:   users.User{Name: "bodie"},
		PWHash: bs[:],
	}), IsNil)
	sesh := s.cli.State.Session
	c.Check(sesh.Expiration, Not(Equals), time.Time{})
	c.Check(sesh.Token, NotNil)
}

func (s *ClientSuite) TestLogin(c *C) {
	var ts []incept.Ticket
	c.Assert(s.cli.CreateTickets(&ts, 1), IsNil)
	c.Check(len(ts), Equals, 1)

	l := new(auth.Login)
	ticket := ts[0].String()
	c.Assert(s.cli.CreateLogin(l, ticket, "bodie", "hello"), IsNil)
	c.Check(l, DeepEquals, &auth.Login{
		User: users.User{
			Name: "bodie",
			Coin: 0,
		},
	})

	bs := sha256.Sum256([]byte("hello"))
	c.Assert(s.cli.Login(&auth.Login{
		User:   users.User{Name: "bodie"},
		PWHash: bs[:],
	}), IsNil)
	sesh := s.cli.State.Session
	c.Check(sesh.Expiration, Not(Equals), time.Time{})
}

func (s *ClientSuite) TestGetProfile(c *C) {
	var ts []incept.Ticket
	c.Assert(s.cli.CreateTickets(&ts, 1), IsNil)
	c.Check(len(ts), Equals, 1)

	l := new(auth.Login)
	ticket := ts[0].String()
	c.Assert(s.cli.CreateLogin(l, ticket, "bodie", "hello"), IsNil)
	c.Check(l, DeepEquals, &auth.Login{
		User: users.User{
			Name: "bodie",
			Coin: 0,
		},
	})

	bs := sha256.Sum256([]byte("hello"))
	c.Assert(s.cli.Login(&auth.Login{
		User:   users.User{Name: "bodie"},
		PWHash: bs[:],
	}), IsNil)

	u := new(users.User)
	c.Assert(s.cli.GetProfile(u), IsNil)
	c.Check(*u, DeepEquals, l.User)
}

func (s *ClientSuite) TestDeleteProfile(c *C) {
	var ts []incept.Ticket
	c.Assert(s.cli.CreateTickets(&ts, 1), IsNil)
	c.Check(len(ts), Equals, 1)

	l := new(auth.Login)
	ticket := ts[0].String()
	c.Assert(s.cli.CreateLogin(l, ticket, "bodie", "hello"), IsNil)
	c.Check(l, DeepEquals, &auth.Login{
		User: users.User{
			Name: "bodie",
			Coin: 0,
		},
	})

	bs := sha256.Sum256([]byte("hello"))
	c.Assert(s.cli.Login(&auth.Login{
		User:   users.User{Name: "bodie"},
		PWHash: bs[:],
	}), IsNil)

	u := new(users.User)
	c.Assert(s.cli.GetProfile(u), IsNil)
	c.Check(*u, DeepEquals, l.User)

	c.Assert(s.cli.DeleteProfile(), IsNil)
	c.Check(s.db.View(users.CheckUserNotExist(u)), IsNil)
}

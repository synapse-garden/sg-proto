package cmd

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/client"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/rest"
	"github.com/synapse-garden/sg-proto/stream"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
	xws "golang.org/x/net/websocket"
)

const help = `SG Help:
  - help                             -- Show this help message
  - quit                             -- Exit SG
  - login <username> <pw>            -- Log into your account
  - incept <ticket> <username> <pw>  -- Create a new user
  - profile                          -- Check profile
  - profile delete                   -- Delete profile
  - stream <username>                -- Open a stream to username
  - admin key <key>                  -- Log in as admin
`

const adminHelp = help + `
  - admin tickets [n]                -- Create n tickets
`

type Command func(*client.Client) error

func Confirm(text string, comm Command) Command {
	return GetInput(
		fmt.Sprintf("really %s? (y/n)", text),
		func(s string) Command {
			switch s {
			case "y", "Y", "yes":
				return comm
			case "n", "N", "no":
				return OutputString("canceled\n")
			default:
				// TODO: improve UX without recursing
				// Confirm(...)(c)
				return OutputStringf("Invalid option "+
					"%#q; input a valid option.\n", s)
			}
		})
}

func GetInput(prompt string, comm func(string) Command) Command {
	return func(c *client.Client) error {
		if err := OutputStringf("%s: ", prompt)(c); err != nil {
			return err
		}
		s := c.State
		if !s.Scan() {
			return OutputString("failed to scan input")(c)
		}
		return comm(s.Text())(c)
	}
}

func OutputFrom(r io.Reader) Command {
	return func(c *client.Client) error {
		_, err := io.Copy(c.State, r)
		return err
	}
}

func OutputString(vals ...interface{}) Command {
	return func(c *client.Client) error {
		_, err := fmt.Fprint(c.State, vals...)
		return err
	}
}

func OutputStringf(f string, vals ...interface{}) Command {
	return func(c *client.Client) error {
		_, err := fmt.Fprintf(c.State, f, vals...)
		return err
	}
}

func OutputJSON(f string, val interface{}) Command {
	return func(c *client.Client) error {
		if _, err := fmt.Fprintln(c.State, f); err != nil {
			return err
		}
		return json.NewEncoder(c.State).Encode(val)
	}
}

func OutputError(err error, f string) Command {
	return func(c *client.Client) error {
		_, err := fmt.Fprint(c.State,
			errors.Wrap(err, f).Error()+"\n")
		return err
	}
}

func OutputErrorf(err error, f string, vals ...interface{}) Command {
	return func(c *client.Client) error {
		_, err := fmt.Fprintf(c.State,
			errors.Wrapf(err, f, vals...).Error())
		return err
	}
}

func OutputHelp(topics ...string) Command {
	return func(c *client.Client) error {
		h := help
		if c.APIKey != "" {
			h = adminHelp
		}
		if len(topics) == 0 {
			return OutputStringf(h)(c)
		}
		switch topics[0] {
		case "login":
			return OutputStringf(h)(c)
		case "stream", "streams":
			return OutputStringf("usage: login <name> <password>\n")(c)
		default:
			return OutputStringf("unknown command %#q\n%s", topics[0], h)(c)
		}
	}
}

func Info(c *client.Client) error {
	info := new(rest.SourceInfo)
	if err := c.Info(info); err != nil {
		return errors.Wrap(err, "failed to get backend /source info")
	}
	return OutputStringf(
		"SG is online:\nLicensed to %s under %s.\nSource "+
			"location: %s\n\n",
		info.LicensedTo,
		info.License,
		info.Location,
	)(c)
}

func Incept(tkt, name, pw string) Command {
	return func(c *client.Client) error {
		if _, err := uuid.FromString(tkt); err != nil {
			return OutputErrorf(err, "invalid ticket %#q", tkt)(c)
		}
		l := new(auth.Login)
		if err := c.CreateLogin(l, tkt, name, pw); err != nil {
			return OutputError(err, "failed to create user")(c)
		}

		return OutputJSON("New user created: ", l.User)(c)
	}
}

func Login(user, pw string) Command {
	return func(c *client.Client) error {
		pwbs := sha256.Sum256([]byte(pw))
		l := &auth.Login{
			User:   users.User{Name: user},
			PWHash: pwbs[:],
		}
		if err := c.Login(l); err != nil {
			return OutputError(err, "failed to create user")(c)
		}
		return OutputStringf("user %s logged in\n", l.Name)(c)
	}
}

func LoggedIn(c *client.Client) error {
	switch {
	case c.State.Session.Token == nil:
		return errors.New("no session")
	case time.Now().After(c.State.Session.Expiration):
		return errors.New("session timed out")
	}
	return nil
}

func Logout(c *client.Client) error {
	if err := LoggedIn(c); err != nil {
		return OutputError(err, "not logged in")(c)
	} else if err := c.Logout(); err != nil {
		return OutputError(err, "failed to log out")(c)
	}
	c.State.Session = new(auth.Session)
	return OutputString("logged out successfully")(c)
}

func DeleteProfile(c *client.Client) error {
	if err := LoggedIn(c); err != nil {
		return OutputError(err, "not logged in")(c)
	} else if err = c.DeleteProfile(); err != nil {
		return OutputError(err, "failed to delete profile")(c)
	}
	c.State.Session = new(auth.Session)
	return OutputString("profile deleted successfully")(c)
}

func Profile(c *client.Client) error {
	if err := LoggedIn(c); err != nil {
		return OutputError(err, "not logged in")(c)
	}
	user := new(users.User)
	if err := c.GetProfile(user); err != nil {
		return OutputError(err, "failed to get profile")(c)
	}

	return OutputJSON("User: ", user)(c)
}

func getStreamByName(c *client.Client, name string) (*stream.Stream, error) {
	strs := new([]*stream.Stream)
	// Find or create a stream for the user.
	if err := c.AllStreams(strs); err != nil {
		return nil, err
	}
	for _, s := range *strs {
		if s.Name == name {
			return s, nil
		}
	}
	return nil, errors.Errorf("stream %#q not found", name)
}

func getStreamByID(c *client.Client, uu uuid.UUID) (*stream.Stream, error) {
	str := new(stream.Stream)
	return str, c.GetStream(str, uu.String())
}

// getStream gets a Stream which has the given name and participants.
// If no such Stream exists, it POSTs and returns a new one for the user.
func getStream(
	c *client.Client,
	name, from, to string,
) (*stream.Stream, error) {
	strs := new([]*stream.Stream)
	// Find or create a stream for the user.
	if err := c.AllStreams(strs); err != nil {
		return nil, err
	}
	for _, s := range *strs {
		if s.Name != name {
			continue
		}

		rsTo := s.Readers[to]
		fromOwner := s.Owner == from
		wsTo, wsFrom := s.Writers[to], s.Writers[from]
		if (wsFrom || fromOwner) && (rsTo || wsTo) {
			return s, nil
		}
	}

	return client.NewStream(c, name, from, to)
}

func Stream(which string, to ...string) Command {
	return func(c *client.Client) error {
		if err := LoggedIn(c); err != nil {
			return OutputError(err, "not logged in")(c)
		}
		user := c.State.User
		if user == nil {
			user = new(users.User)
			if err := c.GetProfile(user); err != nil {
				return OutputError(err, "failed to get profile")(c)
			}
			c.State.User = user
		}

		uu := uuid.FromStringOrNil(which)
		var str *stream.Stream
		var err error

		switch {
		case strings.HasPrefix(which, "#"):
			// User wants a stream by Name
			str, err = getStreamByName(c, which)
		case !uuid.Equal(uu, uuid.Nil):
			// User wants a stream by ID
			str, err = getStreamByID(c, uu)
		default:
			// The passed value was not a UUID or a
			// #chatname, so it was a read / write user's
			// name.
			//
			// User wants to find or create a stream.
			str, err = getStream(c,
				"#chat",
				user.Name, which,
			)
		}

		if err != nil {
			return OutputError(err, "failed to get Stream")(c)
		}
		if err := OutputStringf("joining stream %#q\n", str.ID)(c); err != nil {
			return OutputError(err, "failed to output")(c)
		}

		conn, err := c.GetStreamWS(str.ID)
		if err != nil {
			return OutputError(err, "failed to open websocket")(c)
		}

		s := c.State
		if err := OutputStringf("streaming on %s\n", str.Name)(c); err != nil {
			return err
		}

		errs := make(chan error, 10)
		go func() {
			for s.Scan() {
				err = xws.JSON.Send(conn, &stream.Message{
					Content: s.Text(),
				})
				if err != nil {
					select {
					case errs <- errors.Wrap(err, "send"):
					default:
					}

					return
				}
			}
		}()
		go func() {
			msg := new(stream.Message)
			enc := json.NewEncoder(s)
			for {
				if err := xws.JSON.Receive(conn, msg); err != nil {
					select {
					case errs <- errors.Wrap(err, "recieve"):
					default:
					}

					return
				}
				if err = enc.Encode(msg); err != nil {
					select {
					case errs <- errors.Wrap(err, "encode"):
					default:
					}

					return
				}
			}
		}()
		defer close(errs)
		return <-errs
	}
}

func Quit(_ *client.Client) error {
	return ErrQuit
}

var ErrQuit = errors.New("goodbye")

// GetCommand returns a Command function, including help messages.
func GetCommand(args ...string) Command {
	switch {
	case len(args) < 1:
		return OutputHelp()
	}

	cmd, opts := args[0], args[1:]
	switch cmd {
	case "h", "help":
		return OutputHelp(opts...)
	case "quit", "q", "x", "exit", "bye":
		return Quit
	case "incept":
		if len(opts) != 3 {
			return OutputHelp("incept")
		}
		return Incept(opts[0], opts[1], opts[2])
	case "login":
		switch len(opts) {
		case 1:
			return GetInput("enter password", func(s string) Command {
				return Login(opts[0], s)
			})
		case 2:
			return Login(opts[0], opts[1])
		default:
			return OutputHelp("login")
		}
	case "logout":
		return Logout
	case "admin":
		return AdminCommand(opts...)
	case "do":
		return OutputHelp()
	case "profile":
		switch len(opts) {
		case 0:
			return Profile
		case 1:
			return Confirm("delete profile", DeleteProfile)
		default:
			return OutputHelp(cmd)
		}
	case "stream", "streams":
		if len(opts) != 1 {
			return OutputHelp(cmd)
		}
		return Stream(opts[0])
	default:
		return OutputHelp(cmd)
	}
}

// GetTickets creates n new tickets using the given Admin key.
func GetTickets(n int) Command {
	return func(c *client.Client) error {
		tkts := new([]incept.Ticket)
		if err := c.GetTickets(tkts, n); err != nil {
			return OutputErrorf(err, "failed to get %d ticket(s)", n)(c)
		}

		return OutputJSON("got tickets:\n", tkts)(c)
	}
}

// Tickets creates n new tickets using the given Admin key.
func Tickets(n int) Command {
	return func(c *client.Client) error {
		tkts := new([]incept.Ticket)
		if err := c.CreateTickets(tkts, n); err != nil {
			return OutputErrorf(err, "failed to create %d ticket(s)", n)(c)
		}

		return OutputJSON("created tickets:\n", tkts)(c)
	}
}

func SetAPI(key string) Command {
	return func(c *client.Client) error {
		if bs, err := base64.StdEncoding.DecodeString(key); err != nil {
			return OutputErrorf(err, "invalid admin key %#q", key)(c)
		} else if _, err := uuid.FromBytes(bs); err != nil {
			return OutputErrorf(err, "invalid admin key %#q", key)(c)
		}

		if err := c.VerifyAdmin(key); err != nil {
			return OutputErrorf(err, "invalid admin key %#q", key)(c)
		}

		c.APIKey = key
		return OutputString("API key set")(c)
	}
}

// AdminCommand gets an Admin sub-command.
func AdminCommand(args ...string) Command {
	switch {
	case len(args) < 1:
		return OutputHelp("admin")
	}

	cmd, opts := args[0], args[1:]
	switch cmd {
	case "key", "api-key", "api":
		if len(opts) != 1 {
			return OutputHelp(cmd)
		}
		return SetAPI(opts[0])
	case "ticket", "tickets":
		switch len(opts) {
		case 0:
			return Tickets(1)
		case 1:
			if opts[0] == "get" {
				return GetTickets(10)
			}

			n, err := strconv.Atoi(opts[0])
			switch {
			case err != nil:
				return OutputStringf(
					"must provide integer 'n': %s",
					err.Error(),
				)
			case n < 1:
				return OutputStringf("must provide positive 'n'")

			}
			return Tickets(n)

		case 2:
			if opts[0] != "get" {
				return OutputHelp(cmd)
			}

			n, err := strconv.Atoi(opts[1])
			switch {
			case err != nil:
				return OutputStringf(
					"must provide integer 'n': %s",
					err.Error(),
				)
			case n < 1:
				return OutputStringf("must provide positive 'n'")

			}
			return GetTickets(n)
		}
		fallthrough

	default:
		return OutputHelp(cmd)
	}
}

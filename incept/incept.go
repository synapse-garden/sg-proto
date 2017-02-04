package incept

import (
	"encoding/json"
	"fmt"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	uuid "github.com/satori/go.uuid"
)

var TicketBucket = store.Bucket("tickets")

type ErrTicketMissing string

func (e ErrTicketMissing) Error() string {
	return fmt.Sprintf("no such ticket %#q", string(e))
}

type Ticket uuid.UUID

func (t Ticket) Bytes() []byte  { return uuid.UUID(t).Bytes() }
func (t Ticket) String() string { return uuid.UUID(t).String() }

func (t *Ticket) UnmarshalJSON(bs []byte) error {
	var val string
	if err := json.Unmarshal(bs, &val); err != nil {
		return err
	}
	uu, err := uuid.FromString(val)
	if err != nil {
		return err
	}

	*t = Ticket(uu)
	return nil
}

func (t Ticket) MarshalJSON() ([]byte, error) {
	return json.Marshal(t.String())
}

func NewTickets(ts ...Ticket) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		for _, t := range ts {
			if err := NewTicket(t)(tx); err != nil {
				return err
			}
		}
		return nil
	}
}

func NewTicket(t Ticket) func(*bolt.Tx) error {
	return store.Put(TicketBucket, t.Bytes(), nil)
}

func CheckTicketExist(key Ticket) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		err := store.CheckExists(TicketBucket, key.Bytes())(tx)
		if store.IsMissing(err) {
			return ErrTicketMissing(key.String())
		}
		return err
	}
}

func DeleteTickets(ts ...Ticket) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		for _, t := range ts {
			err := store.Delete(TicketBucket, t.Bytes())(tx)
			if err != nil {
				return err
			}
		}
		return nil
	}
}

func PunchTicket(key Ticket) func(*bolt.Tx) error {
	return store.Delete(TicketBucket, key.Bytes())
}

// Incept checks that the given Ticket exists, and that the given User
// does not (by name.)  Then it tries to create the given user,
// and delete the given key.  Any error will cause this to roll back.
func Incept(
	key Ticket,
	l *auth.Login,
	db *bolt.DB,
) error {
	user := &(l.User)
	name := user.Name
	// Check if given ticket exists (nil => it exists)
	if err := db.View(store.Wrap(
		CheckTicketExist(key),
		users.CheckNotExist(name),
		auth.CheckLoginNotExist(l),
	)); err != nil {
		return err
	}
	// Create user or fail
	if err := db.Update(store.Wrap(
		CheckTicketExist(key),
		users.CheckNotExist(name),
		auth.CheckLoginNotExist(l),
		PunchTicket(key),
		users.Create(user),
		auth.Create(l, uuid.NewV4()),
	)); err != nil {
		return err
	}

	return nil
}

// InceptNoTicket is a method for admins to create a user without
// punching a Ticket.
func InceptNoTicket(l *auth.Login, db *bolt.DB) error {
	user := &(l.User)
	if err := db.View(store.Wrap(
		users.CheckNotExist(user.Name),
		auth.CheckLoginNotExist(l),
	)); err != nil {
		return err
	}

	// Create user or fail
	if err := db.Update(store.Wrap(
		users.CheckNotExist(user.Name),
		auth.CheckLoginNotExist(l),
		users.Create(user),
		auth.Create(l, uuid.NewV4()),
	)); err != nil {
		return err
	}

	return nil
}

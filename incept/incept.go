package incept

import (
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
	// Check if given ticket exists (nil => it exists)
	if err := db.View(store.Wrap(
		CheckTicketExist(key),
		users.CheckUserNotExist(user),
		auth.CheckLoginNotExist(l),
	)); err != nil {
		return err
	}
	// Create user or fail
	if err := db.Update(store.Wrap(
		CheckTicketExist(key),
		users.CheckUserNotExist(user),
		auth.CheckLoginNotExist(l),
		PunchTicket(key),
		users.Create(&(l.User)),
		auth.Create(l, uuid.NewV4()),
	)); err != nil {
		return err
	}

	return nil
}

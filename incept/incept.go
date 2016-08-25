package incept

import (
	"encoding/json"
	"io"
	"time"

	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	uuid "github.com/satori/go.uuid"
)

var (
	TicketBucket = store.Bucket("tickets")

	masterUUID uuid.UUID
)

func init() {
	masterUUID = uuid.NewV4()
}

type Ticket uuid.UUID

func (t Ticket) Bytes() []byte  { return uuid.UUID(t).Bytes() }
func (t Ticket) String() string { return uuid.UUID(t).String() }

func NewTicket(tx *bolt.Tx) (Ticket, error) {
	u := uuid.NewV5(masterUUID, time.Now().String())
	if err := store.Put(TicketBucket, u.Bytes(), nil)(tx); err != nil {
		return Ticket(uuid.Nil), err
	}
	return Ticket(u), nil
}

func CheckTicket(key Ticket) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		bs := tx.Bucket(TicketBucket).Get(key.Bytes())
		if bs == nil {
			return store.MissingError(key.Bytes())
		}
		return nil
	}
}

func DeleteKey(key Ticket) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		return tx.Bucket(TicketBucket).Delete(key.Bytes())
	}
}

// Incept checks that the given Ticket exists, and that the given User
// does not (by name.)  Then it tries to create the given user,
// and delete the given key.  Any error will cause this to roll back.
func Incept(
	w io.Writer,
	key Ticket,
	u *users.User,
	db *bolt.DB,
) error {
	kbs, name := key.Bytes(), []byte(u.Name)
	// Check if given ticket exists (nil => it exists)
	if err := db.View(store.Wrap(
		store.CheckExists(TicketBucket, kbs),
		store.CheckNotExist(users.UserBucket, name),
	)); err != nil {
		return err
	}
	// Create user or fail
	if err := db.Update(store.Wrap(
		store.Delete(TicketBucket, kbs),
		users.Create(u),
	)); err != nil {
		return err
	}

	return json.NewEncoder(w).Encode(u)
}

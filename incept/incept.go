package incept

import (
	"fmt"
	"io"
	"time"

	"github.com/synapse-garden/sg-proto/user"

	"github.com/boltdb/bolt"
	uuid "github.com/satori/go.uuid"
)

var (
	TicketBucket = []byte("tickets")

	masterUUID uuid.UUID
)

func init() {
	masterUUID = uuid.NewV4()
}

type MissingError []byte

func (m MissingError) Error() string {
	return fmt.Sprintf("no such ticket %#q", string(m))
}

func IsMissing(err error) bool {
	_, ok := err.(MissingError)
	return ok
}

type Ticket []byte

func NewTicket(tx *bolt.Tx) error {
	u := uuid.NewV5(masterUUID, time.Now().String())
	return tx.Bucket(TicketBucket).Put(u.Bytes(), nil)
}

func CheckKey(key Ticket) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		if len(tx.Bucket(TicketBucket).Get(key)) == 0 {
			return MissingError(key)
		}

		return nil
	}
}

func DeleteKey(key Ticket) func(*bolt.Tx) error {
	return func(tx *bolt.Tx) error {
		return tx.Bucket(TicketBucket).Delete(key)
	}
}

func Incept(
	w io.Writer,
	key Ticket,
	u *user.User,
	db *bolt.DB,
) error {
	// Check if given ticket exists (nil => it exists)
	if err := db.View(CheckKey(key)); err != nil {
		return err
	}
	// Create user or fail
	if err := db.Update(func(tx *bolt.Tx) error {
		if err := user.Create(u)(tx); err != nil {
			return err
		}
		if err := DeleteKey(key)(tx); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}

	return nil
}

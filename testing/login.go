package testing

import (
	"time"

	"github.com/synapse-garden/sg-proto/auth"
	"github.com/synapse-garden/sg-proto/incept"
	"github.com/synapse-garden/sg-proto/users"

	"github.com/boltdb/bolt"
	"github.com/pkg/errors"
	uuid "github.com/satori/go.uuid"
)

// MakeLogin creates a new auth.Login in the DB using the given name and
// SHA256-hashed password.
func MakeLogin(name, pw string, db *bolt.DB) (*auth.Login, error) {
	tick := incept.Ticket(uuid.NewV4())
	if err := db.Update(incept.NewTickets(tick)); err != nil {
		return nil, errors.Wrapf(err, "failed to create ticket for user %#q", name)
	}

	user := &auth.Login{
		User:   users.User{Name: name},
		PWHash: Sha256(pw),
	}

	if err := incept.Incept(tick, user, db); err != nil {
		return nil, errors.Wrapf(err, "failed to incept user %#q", name)
	}

	return user, nil
}

// GetSession creates a new Session in the DB for the given User, if the
// user has a valid login.
func GetSession(userID string, sesh *auth.Session, db *bolt.DB) error {
	return db.Update(auth.NewSession(
		sesh,
		time.Now().Add(auth.Expiration),
		auth.Expiration,
		auth.NewToken(auth.BearerType),
		auth.NewToken(auth.RefreshType),
		userID,
	))
}

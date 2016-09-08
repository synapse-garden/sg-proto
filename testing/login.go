package testing

import (
	"time"

	"github.com/synapse-garden/sg-proto/auth"

	"github.com/boltdb/bolt"
)

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

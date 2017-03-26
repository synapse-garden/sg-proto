package testing

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/synapse-garden/sg-proto/auth"

	"github.com/boltdb/bolt"
)

func FindSession(db *bolt.DB, expiration time.Time) (*auth.Session, error) {
	ret := new(auth.Session)
	err := db.View(func(tx *bolt.Tx) error {
		s := new(auth.Session)
		err := tx.Bucket(auth.SessionBucket).ForEach(
			func(_, v []byte) error {
				if err := json.Unmarshal(v, s); err != nil {
					return err
				}
				if s.Expiration == expiration {
					*ret = *s
				}
				return nil
			},
		)

		if err != nil {
			return err
		}

		if ret.Expiration != expiration {
			return fmt.Errorf(
				"session at %s not found",
				expiration.String(),
			)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	return ret, nil
}

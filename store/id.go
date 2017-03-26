package store

import (
	"encoding/json"

	uuid "github.com/satori/go.uuid"
)

// ID is a JSON-serializable UUID.
type ID uuid.UUID

// HashWith returns a V5 UUID (defined as the SHA-1 of the given ID and
// the given input string) as an ID.
func (i ID) HashWith(some string) ID {
	return ID(uuid.NewV5(uuid.UUID(i), some))
}

// MarshalJSON implements json.Marshalder on ID.
func (i ID) MarshalJSON() ([]byte, error) {
	return json.Marshal(uuid.UUID(i).String())
}

// UnmarshalJSON implements json.Unmarshaler on ID.
func (i *ID) UnmarshalJSON(from []byte) error {
	into := new(string)
	if err := json.Unmarshal(from, into); err != nil {
		return err
	} else if id, err := uuid.FromString(*into); err != nil {
		return err
	} else {
		*i = ID(id)
		return nil
	}
}

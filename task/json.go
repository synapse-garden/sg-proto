package task

import (
	"encoding/json"

	"github.com/synapse-garden/sg-proto/store"
)

func (i ID) MarshalJSON() ([]byte, error) {
	return store.ID(i).MarshalJSON()
}

func (i *ID) UnmarshalJSON(from []byte) error {
	into := new(store.ID)
	if err := json.Unmarshal(from, into); err != nil {
		return err
	}
	*i = ID(*into)
	return nil
}

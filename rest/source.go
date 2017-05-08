package rest

import (
	"encoding/json"
	"net/http"

	"github.com/synapse-garden/sg-proto/store"

	htr "github.com/julienschmidt/httprouter"
)

// SourceInfo represents the SG-Proto source location and license, and
// implements API.
type SourceInfo struct {
	Version    store.Version `json:"version"`
	License    string        `json:"license"`
	LicensedTo string        `json:"licensedTo"`
	Location   string        `json:"location"`
}

// Bind implements API.Bind on SourceInfo.
func (s SourceInfo) Bind(r *htr.Router) (Cleanup, error) {
	bs, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return nil, err
	}
	r.GET("/source", func(
		w http.ResponseWriter,
		r *http.Request,
		_ htr.Params,
	) {
		w.Write(bs)
	})
	return nil, nil
}

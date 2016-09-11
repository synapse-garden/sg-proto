package rest

import (
	"encoding/json"
	"net/http"

	"github.com/boltdb/bolt"
	htr "github.com/julienschmidt/httprouter"
)

type SourceInfo struct {
	License    string `json:"license"`
	LicensedTo string `json:"licensedTo"`
	Location   string `json:"location"`
}

func Source(source *SourceInfo) API {
	bs, err := json.MarshalIndent(source, "", "  ")
	return func(r *htr.Router, _ *bolt.DB) error {
		if err != nil {
			return err
		}
		r.GET("/source", func(
			w http.ResponseWriter,
			r *http.Request,
			_ htr.Params,
		) {
			w.Write(bs)
		})
		return nil
	}
}

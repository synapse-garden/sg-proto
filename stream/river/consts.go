package river

import "github.com/synapse-garden/sg-proto/store"

// Survey is a survey request constant.
type Survey []byte

// Response is a response request constant (may be suffixed with an ID.)
type Response []byte

var (
	// Byte request / response constants.
	HUP     = Survey("HUP")
	OK      = Response("OK")
	UNKNOWN = Response("IDK")

	// RiverBucket stores Rivers and their users.  Buckets in RiverBucket
	// correspond to Streams from StreamBucket by ID, and every River ID in
	// the bucket corresponds to a connected River.
	RiverBucket = store.Bucket("rivers")

	// HangupBucket is where Respondents are stored.  Pass this to
	// NewSurvey as the first Bucket argument for most Surveys.
	HangupBucket = store.Bucket("hangups")
)

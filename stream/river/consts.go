package river

// Survey is a survey request constant.
type Survey []byte

// Response is a response request constant (may be suffixed with an ID.)
type Response []byte

// Byte request / response constants.
var (
	HUP     = Survey("HUP")
	OK      = Response("OK")
	UNKNOWN = Response("IDK")
)

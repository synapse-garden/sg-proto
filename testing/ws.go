package testing

import (
	"net/url"

	ws "golang.org/x/net/websocket"
)

// GetWSClient opens and returns a websocket.Conn to the given URL using
// the given auth token.
func GetWSClient(token, urlStr string) (*ws.Conn, error) {
	urlLoc, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	urlLoc.Scheme = "ws"

	return ws.DialConfig(&ws.Config{
		Location: urlLoc,
		Origin:   &url.URL{},
		Version:  ws.ProtocolVersionHybi13,
		Protocol: []string{"Bearer+" + token},
	})
}

package notif

import (
	js "encoding/json"

	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream/river"
)

// Encoder is a type implementing an Encode to a River.
// TODO: Offer Protobuf / Msgpack Encoder
// TODO: Offer pb / json / mpack Decoder / Translator.
type Encoder interface {
	Encode(river.Pub, store.Resourcer, river.Topic) error
}

type json struct{}

// DefaultEncoder implements a shortcut for JSON encoding to a River on
// a Topic.
var DefaultEncoder = json{}

// Encode implements Encoder on json for DefaultEncoder.
func (json) Encode(r river.Pub, val store.Resourcer, t UserTopic) error {
	bs, err := js.Marshal(val)
	if err != nil {
		return err
	}
	boxBs, err := js.Marshal(store.ResourceBox{
		Name:     val.Resource(),
		Contents: string(bs),
	})
	if err != nil {
		return err
	}
	return r.Send(river.BytesFor(t, boxBs))
}

// Encode uses the default (JSON) Encoder to send the given value on r,
// prefixed with t.
func Encode(r river.Pub, val store.Resourcer, t UserTopic) error {
	return DefaultEncoder.Encode(r, val, t)
}

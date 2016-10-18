package river

// Topic is a Pub/Sub topic.  Package users should export and manage
// their own Topic constants.  Note that this means collisions can
// happen if the package user does not take care to review imported
// Topics.
type Topic interface {
	// Prefix must be prepended to Code.  This permits a quick match
	// on message prefix.
	Prefix() byte
	Code() []byte
	Name() string
	Len() int
}

// GlobalTopic is a Subscribe Topic which messages not belonging to any
// other Topic should be sent on.
type GlobalTopic struct{}

// Prefix returns the GlobalTopic Prefix, 0.
func (GlobalTopic) Prefix() byte { return 0 }

// Code implements Topic.Code on GlobalTopic.
func (GlobalTopic) Code() []byte { return []byte{0} }

// Name implements Topic.Name on GlobalTopic, returning "all".
func (GlobalTopic) Name() string { return "global" }

// Len implements Topic.Len on GlobalTopic, returning 1.
func (GlobalTopic) Len() int { return 1 }

// Global is a package constant for GlobalTopic.
var Global = GlobalTopic{}

// BytesFor returns the bytes for a given []byte on the given Topic.
func BytesFor(t Topic, content []byte) []byte {
	return append(t.Code(), content...)
}

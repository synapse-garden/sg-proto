package notif

import (
	"encoding/base64"

	"github.com/synapse-garden/sg-proto/stream"

	blake2b "github.com/minio/blake2b-simd"
)

// Topic consts.
const (
	UserPrefix    = byte(1)
	userTopicSize = 65
)

// UserTopic is a Topic made from a unique string.
type UserTopic [64]byte

// MakeUserTopic calculates a UserTopic by appending a 64-byte truncated
// BLAKE2 hash of the given id to "USER".
func MakeUserTopic(id string) UserTopic {
	return UserTopic(blake2b.Sum512([]byte(id)))
}

// Prefix implements Topic.Prefix on UserTopic.
func (t UserTopic) Prefix() byte { return UserPrefix }

// Code implements Topic.Code on UserTopic.
func (t UserTopic) Code() []byte {
	return append([]byte{UserPrefix}, t[:]...)
}

// Name implements Topic.Name on UserTopic.
func (t UserTopic) Name() string {
	return "user " + base64.StdEncoding.EncodeToString(t[:])
}

// Len implements Topic.Len on UserTopic.
func (t UserTopic) Len() int { return userTopicSize }

// Topics gets the stream.Topics for the given User.  This is currently
// a single UserTopic for the user.
func Topics(userID string) []stream.Topic {
	return []stream.Topic{MakeUserTopic(userID)}
}

package convo

import (
	"bytes"
	"encoding/json"
	"time"

	"github.com/synapse-garden/sg-proto/store"

	"github.com/boltdb/bolt"
)

// Message is a container for a convo message.
type Message struct {
	Sender    string    `json:"sender"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
}

// GetMessageRange gets a slice of up to max Messages for the given
// time range in the given Convo.
func GetMessageRange(
	convoID string,
	from, to time.Time,
	max int,
	tx *bolt.Tx,
) ([]Message, error) {
	b, err := store.GetNestedBucket(
		tx.Bucket(MessageBucket),
		store.Bucket(convoID),
	)
	switch {
	case store.IsMissingBucket(err):
		return nil, errMissing(err.(store.ErrMissingBucket))
	case err != nil:
		return nil, err
	}

	c := b.Cursor()
	fromBytes := from.AppendFormat(nil, time.RFC3339)
	toBytes := to.AppendFormat(nil, time.RFC3339)
	t, msg := c.Seek(toBytes)
	switch {
	case t != nil && bytes.Compare(t, toBytes) > 0:
		// Seek gave us something too new.  Try previous.
		if t, msg = c.Prev(); t == nil {
			// It was the first item.
			return []Message{}, nil
		} else if bytes.Compare(t, fromBytes) < 0 {
			// Something older was found, but too old.
			return []Message{}, nil
		}
	case t == nil:
		// Nothing that new exists at all.  Try previous.
		if t, msg = c.Prev(); t == nil {
			// There were no messages in the bucket at all.
			return []Message{}, nil
		} else if bytes.Compare(t, fromBytes) < 0 {
			// There were previous messages, but too old.
			return []Message{}, nil
		}
	}
	// We now have our cursor at the end of our range.

	var result []Message
	for t != nil && len(result) < max && bytes.Compare(t, fromBytes) >= 0 {
		next := Message{}
		if err := json.Unmarshal(msg, &next); err != nil {
			return nil, err
		}
		result = append(result, next)
		t, msg = c.Prev()
	}

	// Since we iterated backwards, reverse the array now.
	lenRes := len(result)
	mp := lenRes / 2
	for i := 0; i < mp; i++ {
		result[lenRes-i-1], result[i] = result[i], result[lenRes-i-1]
	}

	return result, nil
}

// GetMessages gets a slice of up to 10 Messages for the last week in
// the given Convo.
func GetMessages(convoID string, tx *bolt.Tx) ([]Message, error) {
	now := time.Now()
	then := now.Add(-7 * 24 * time.Hour)
	return GetMessageRange(convoID, now, then, 50, tx)
}

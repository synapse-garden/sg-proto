package convo

import (
	"encoding/binary"
	"encoding/json"
	"log"
	"time"

	"github.com/pkg/errors"
	"github.com/synapse-garden/sg-proto/rest/ws"
	"github.com/synapse-garden/sg-proto/store"
	"github.com/synapse-garden/sg-proto/stream/river"

	"github.com/boltdb/bolt"
)

// Frequency is the frequency with which the Scribe writes to the log.
const Frequency = 1 * time.Second

// MaxBuffer is the maximum number of messages which will be buffered
// before writing them to disk.
const MaxBuffer = 128

// Scribe is a BUS consumer which does nothing but log received messages
// to the Messages bucket under its ID (which is the same ID as the
// stream it belongs to.)
type Scribe string

// CheckExists is a bolt View function which returns nil if the
// given Scribe is already running.  The Scribe should have the same ID
// as the Stream it is logging.  If the Scribe does not exist, this
// returns errNotExists.
func (s Scribe) CheckExists(tx *bolt.Tx) error {
	b, err := store.GetNestedBucket(
		tx.Bucket(river.RiverBucket),
		ScribeBucket,
		store.Bucket(s),
	)
	switch {
	case store.IsMissingBucket(err):
		return errMissing(err.(store.ErrMissingBucket))
	case err != nil:
		return err
	}

	if k, _ := b.Cursor().First(); k == nil {
		return errMissing(s)
	}
	return nil
}

// Entry is a Scribe log entry.  Its key is an RFC3339 timestamp and its
// value is the JSON representation of the message.
type Entry struct {
	key, value []byte
}

// Checkin is a bolt Update function which a sequence ID, and true, if
// the caller is the first caller for the given Scribe.  If this returns
// true, the caller should also use Scribe.Spawn to start a new Scribe.
func (s Scribe) Checkin(tx *bolt.Tx) (uint64, bool, error) {
	scrB, err := store.MakeNestedBucket(
		tx.Bucket(river.RiverBucket),
		ScribeBucket,
		store.Bucket(s),
	)
	if err != nil {
		return 0, false, err
	}

	seq := scrB.Sequence()
	seqBs := make([]byte, 8)
	binary.LittleEndian.PutUint64(seqBs, seq)

	// If anyone else exists, return false.
	k, _ := scrB.Cursor().First()
	return seq, k == nil, scrB.Put(seqBs, nil)
}

// Checkout is a bolt Update function which takes a sequence ID, and
// returns true if it is the last to check out.  If it is the last to
// check out, the caller should also call Scribe.Hangup.
func (s Scribe) Checkout(id uint64, tx *bolt.Tx) (bool, error) {
	scrB, err := store.GetNestedBucket(
		tx.Bucket(river.RiverBucket),
		ScribeBucket,
		store.Bucket(s),
	)
	if err != nil {
		return false, err
	}

	seqBs := make([]byte, 8)
	binary.LittleEndian.PutUint64(seqBs, id)

	// First delete our entry.
	if err := scrB.Delete(seqBs); err != nil {
		return false, err
	}

	// If anything remains after deleting, return false.
	k, _ := scrB.Cursor().First()
	return k == nil, nil
}

// Spawn is a Bolt Update function which creates a new Scribe Bus,
// listening to the given stream in a goroutine.  It also creates a
// Responder, so the Scribe can be hung up.  To hang it up and clean it
// from the database, use Scribe.Hangup.
//
// TODO: Tighten this up using DB funcs in message.go.
func (s Scribe) Spawn(tx *bolt.Tx) error {
	rsp, err := river.NewResponder(tx,
		river.HangupBucket,
		ScribeBucket,
		store.Bucket(s),
	)
	if err != nil {
		return err
	}

	scr, err := river.NewBus(string(s), string(s), tx)
	if err != nil {
		return err
	}

	h := ws.MakeHangupRecver(rsp, scr)
	errCh := make(chan error)
	toScribe := make(chan []byte, 1024)

	go func() { errCh <- river.AwaitHangup(h) }()

	go func(db *bolt.DB) {
		timer := time.NewTimer(Frequency)
	readLoop:
		for {
			var buf []Entry

			// Loop until buffer full or timeout with
			// non-empty buffer, whichever comes first.
		readBuf:
			for len(buf) < MaxBuffer {
				var msg Message
				select {
				case <-timer.C:
					timer.Reset(Frequency)
					if len(buf) > 0 {
						break readBuf
					}
				case bs, ok := <-toScribe:
					if !ok {
						// Input channel closed.
						break readLoop
					}
					err := json.Unmarshal(bs, &msg)
					if err != nil {
						log.Fatalf("Scribe failed to read message %#q: %s", bs, err.Error())
					}

					buf = append(buf, Entry{
						key:   msg.Timestamp.AppendFormat(nil, time.RFC3339),
						value: bs,
					})
				}
			}

			err := db.Update(func(tx *bolt.Tx) error {
				b, e := store.MakeNestedBucket(
					tx.Bucket(MessageBucket),
					store.Bucket(s),
					// TODO: Block buckets
					// TODO: Dated block index?
					// TODO: Messages ordered by ID?
				)
				if e != nil {
					return e
				}

				for _, entry := range buf {
					err := b.Put(entry.key, entry.value)
					if err != nil {
						return err
					}
				}
				return nil
			})
			if err != nil {
				log.Fatalf("Scribe failed to write messages: %s", err.Error())
			}

			timer.Reset(Frequency)
		}

		// After the close survey has finished, the Scribe can
		// be cleaned up.
		<-errCh
		err := db.Update(func(tx *bolt.Tx) error {
			eB := river.DeleteBus(string(s), string(s), scr.ID())(tx)
			eS := river.DeleteResp(tx, h.ID(),
				river.HangupBucket,
				ScribeBucket,
				store.Bucket(s),
			)
			switch {
			case eB != nil && eS != nil:
				return errors.Wrap(eS, eB.Error())
			case eB != nil:
				return eB
			case eS != nil:
				return eS
			default:
				return nil
			}
		})
		if err != nil {
			log.Fatalf("failed to clean up Scribe: %s", err.Error())
		}
	}(tx.DB())

	recver := h.Recver()

	go func() {
		// Recv from r; pass result to the Scribe if no error.
		for bs, err := recver.Recv(); err == nil; bs, err = recver.Recv() {
			toScribe <- bs
		}

		// Once hung up, close the scribe channel, causing the
		// write loop to finish and clean up.
		close(toScribe)
	}()

	return nil
}

// Hangup closes the Scribe.
func (s Scribe) Hangup(db *bolt.DB) error {
	var surv river.Surveyor
	err := db.Update(func(tx *bolt.Tx) (e error) {
		surv, e = river.NewSurvey(tx,
			10*time.Millisecond,
			river.HangupBucket,
			ScribeBucket,
			store.Bucket(s),
		)
		return
	})
	if err != nil {
		return err
	}

	return river.MakeSurvey(surv, river.HUP, river.OK)
}

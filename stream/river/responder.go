package river

import (
	"bytes"
	"encoding/binary"
	"strconv"

	"github.com/synapse-garden/sg-proto/store"

	"github.com/boltdb/bolt"
	"github.com/go-mangos/mangos"
	mg_resp "github.com/go-mangos/mangos/protocol/respondent"
	"github.com/go-mangos/mangos/transport/inproc"
	"github.com/pkg/errors"
)

// Wait waits on the given Survey and responds with the given Response.
// If the Survey is not recognized, it responds with "or".
func Wait(r Responder, surv Survey, rsp, or Response) error {
	bs, err := r.Recv()
	if err != nil {
		return err
	}

	if bytes.Equal(bs, surv) {
		return r.Send(rsp)
	} else if err := r.Send(or); err != nil {
		// If we managed to send our 'or' response, we tell the
		// package user that the received survey was unknown;
		// otherwise we notify that the send failed.  The user
		// may want to retry.
		return err
	} else {
		return errUnknownSurvey(bs)
	}
}

// AwaitHangup waits for HUP, and replies with OK or UNKNOWN suffixed
// with ID, returning any error.
func AwaitHangup(r Responder) error {
	idBs := make([]byte, 8)
	binary.LittleEndian.PutUint64(idBs, r.ID())
	return Wait(r, HUP, append(OK, idBs...), append(UNKNOWN, idBs...))
}

// Responder is a River which implements the RESPONDENT end of the
// SURVEYOR/RESPONDENT scalable protocol.
type Responder interface {
	Send([]byte) error
	Recv() ([]byte, error)
	Close() error

	ID() uint64
}

// respondent is an implementation of Responder storing its ID.
type respondent struct {
	mangos.Socket
	id uint64
}

// NewResponder creates a new SURVEYOR/RESPONDENT respondent nested
// bucket for the given bucket names, if it does not exist.  It assigns
// a new address to a new Responder River based on its path and stores
// it in the innermost bucket.  It then starts listening and returns the
// Responder.
func NewResponder(
	tx *bolt.Tx,
	buckets ...store.Bucket,
) (rsp Responder, e error) {
	var none respondent
	b, err := store.MakeNestedBucket(tx.Bucket(RiverBucket), buckets...)
	if err != nil {
		return none, errors.Wrap(err, "failed to create buckets")
	}

	sock, err := mg_resp.NewSocket()
	if err != nil {
		return none, errors.Wrap(err, "failed to create socket")
	}
	sock.AddTransport(inproc.NewTransport())

	seq, err := b.NextSequence()
	if err != nil {
		return none, errors.Wrap(err, "failed to get next sequence")
	}

	bs := make([]byte, 8)
	binary.LittleEndian.PutUint64(bs, seq)
	uintStr := strconv.FormatUint(seq, 10)

	addrBuf := bytes.NewBufferString("inproc://")
	for _, bucket := range buckets {
		if _, err = addrBuf.Write(append(bucket, '/')); err != nil {
			return none, err
		}
	}
	addrBuf.WriteString(uintStr)

	if err = b.Put([]byte(uintStr), nil); err != nil {
		return none, errors.Wrap(err, "failed to insert River")
	}

	addr := addrBuf.String()

	err = sock.Listen(addr)
	switch {
	case err == mangos.ErrAddrInUse:
		return none, errExists(addr)
	case err != nil:
		return none, errors.Wrap(err, "failed to start listening")
	}

	return respondent{Socket: sock, id: seq}, nil
}

// ID implements Responder.ID on respondent.
func (r respondent) ID() uint64 { return r.id }

func DeleteResp(tx *bolt.Tx, id uint64, buckets ...store.Bucket) error {
	b, err := store.GetNestedBucket(tx.Bucket(RiverBucket), buckets...)
	if err != nil {
		return errors.Wrap(err, "failed to create buckets")
	}
	switch {
	case store.IsMissingBucket(err):
		return errStreamMissing(err.(store.ErrMissingBucket))
	case err != nil:
		return err
	}

	seqBs := []byte(strconv.FormatUint(id, 10))
	return b.Delete(seqBs)
}

package river

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/synapse-garden/sg-proto/store"

	"github.com/boltdb/bolt"
	"github.com/go-mangos/mangos"
	mg_surv "github.com/go-mangos/mangos/protocol/surveyor"
	"github.com/go-mangos/mangos/transport/inproc"
	"github.com/pkg/errors"
)

// DefaultTimeout is the default timeout for Surveys.
var DefaultTimeout = 30 * time.Millisecond

// MakeSurvey sends the given Survey to all connected Responders and
// awaits the expected Response, suffixed with binary.LittleEndian
// uint64 ID.  It retries as necessary up to three times, keeping track
// of the seen IDs of respondents.
func MakeSurvey(s Surveyor, what Survey, expect Response) error {
	var (
		seen       = make(map[uint64]bool)
		expectSeen = s.Expect()
		errs       = make(map[uint64]error)
		num        = len(expectSeen)

		// Surv/Resp const byte lengths
		lr = len(expect)
		lu = len(UNKNOWN)
	)

	// 3 retries
	for i := 0; i < 3; i++ {
		toRecv := num - len(seen) - len(errs)

		// Send the survey.
		if err := s.Send(what); err != nil {
			return err
		}

		// Wait for responses from respondents which have not
		// yet replied.
	query:
		for j := 0; j < toRecv; j++ {
			bs, err := s.Recv()
			switch {
			case err == mangos.ErrProtoState:
				// Nothing was received.  Stop.
				break query
			case err != nil:
				// Some unexpected problem occurred.
				return err
			}
			switch {
			case bytes.HasPrefix(bs, expect):
				// We saw one of our expected responses.
				id := binary.LittleEndian.Uint64(bs[lr:])
				if !expectSeen[id] {
					return errors.Errorf(
						"unexpected respondent %d",
						id,
					)
				}
				seen[id] = true
			case bytes.HasPrefix(bs, UNKNOWN):
				// Someone didn't know what our survey
				// meant.  Tell the caller who.
				id := binary.LittleEndian.Uint64(bs[lr:])
				errs[id] = errUnknownSurvey(bs[lu:])
			default:
				// Something went unexpectedly wrong.
				// Just stop and return the unexpected
				// value.  Some responders might be dead
				// now.
				// TODO: think about this condition.
				return errUnknownSurvey(bs)
			}
		}

		switch {
		case len(seen) == num:
			// We saw all expected responses.  No retry.
			return nil
		case len(seen)+len(errs) == num:
			// Everyone responded, but there were some errors.
			// TODO: serializable error type.
			var rspMsgs []string
			for id, err := range errs {
				rspMsgs = append(rspMsgs, fmt.Sprintf(
					"client %s: %s",
					strconv.FormatUint(id, 10),
					err.Error(),
				))
			}
			return errors.Errorf("errors: %s", strings.Join(
				rspMsgs, ", ",
			))
		}
	}

	var unseen []uint64
	for k := range expectSeen {
		if !seen[k] {
			unseen = append(unseen, k)
		}
	}

	return errors.Errorf("no response from client(s): %+v", unseen)
}

// Surveyor is a River which implements the Surveyor end of the
// SURVEYOR/RESPONDENT scalable protocol.
type Surveyor interface {
	Recv() ([]byte, error)
	Send([]byte) error
	Close() error

	Expect() map[uint64]bool
}

type surveyor struct {
	mangos.Socket

	clients map[uint64]bool
}

// NewSurvey is a View function which returns a Surveyor to run the
// given Survey against all Responders found in the given bucket in
// RiverBucket, with the contents of req. and the given prefix []byte.
// Rivers/responders/:user_id/{1,2,3}
//
// Timeout sets the retry timeout for the underlying mangos.Socket.
func NewSurvey(
	tx *bolt.Tx,
	timeout time.Duration,
	buckets ...store.Bucket,
) (Surveyor, error) {
	var none surveyor
	b, err := store.GetNestedBucket(tx.Bucket(RiverBucket), buckets...)
	switch {
	case store.IsMissingBucket(err) && len(buckets) > 0:
		return none, errStreamMissing([]byte(
			err.(store.ErrMissingBucket),
		))
	case err != nil:
		return none, errors.Wrap(err, "failed to get bucket")
	}

	sock, err := mg_surv.NewSocket()
	if err != nil {
		return none, errors.Wrap(err, "failed to create socket")
	}
	sock.AddTransport(inproc.NewTransport())
	if err = sock.SetOption(mangos.OptionSurveyTime, timeout); err != nil {
		return none, errors.Wrap(err, "failed to set socket timeout")
	}

	addrBuf := bytes.NewBufferString("inproc://")
	for _, bucket := range buckets {
		if _, err = addrBuf.Write(append(bucket, '/')); err != nil {
			return none, err
		}
	}
	expect := make(map[uint64]bool)
	addrStr := addrBuf.String()
	err = b.ForEach(func(k, _ []byte) error {
		keyStr := string(k)
		id, err := strconv.ParseUint(keyStr, 10, 64)
		if err != nil {
			return err
		}
		expect[id] = true

		addr := addrStr + keyStr
		err = sock.Dial(addr)
		switch {
		case err == mangos.ErrAddrInUse:
			return errExists(addr)
		case err != nil:
			return errors.Wrap(err, "failed to start listening")
		default:
			return nil
		}
	})

	return surveyor{Socket: sock, clients: expect}, nil
}

func (s surveyor) Expect() map[uint64]bool { return s.clients }

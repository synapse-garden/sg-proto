package stream

import "fmt"

func MakeMissingErr(id []byte) error {
	return errStreamMissing(id)
}

func MakeExistsErr(id []byte) error {
	return errStreamExists(id)
}

func MakeUnauthorizedErr(user string) error {
	return errUnauthorized(user)
}

func MakeRiverExistsErr(id string) error {
	return errRiverExists(id)
}

func MakeSub(r SubRiver, ts ...Topic) sub {
	if len(ts) == 0 {
		ts = []Topic{Global}
	}
	topics := make(map[byte]Topic)
	for _, t := range ts {
		b := t.Prefix()
		if tb, ok := topics[b]; ok {
			panic(fmt.Sprintf(
				"bad topic %#q: %#q already used",
				t.Name(),
				tb.Name(),
			))
		}
		topics[b] = t
	}
	return sub{
		SubRiver: r,
		topics:   topics,
	}
}

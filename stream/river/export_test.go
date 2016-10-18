package river

import "fmt"

func MakeRiverExistsErr(id string) error {
	return errExists(id)
}

func MakeStreamMissingErr(id []byte) error {
	return errStreamMissing(id)
}

func MakeStreamExistsErr(id []byte) error {
	return errStreamExists(id)
}

func MakeSub(r Sub, ts ...Topic) sub {
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
		Sub:    r,
		topics: topics,
	}
}

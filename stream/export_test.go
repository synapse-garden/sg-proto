package stream

func MakeMissingErr(id []byte) error {
	return errStreamMissing(id)
}

func MakeExistsErr(id []byte) error {
	return errStreamExists(id)
}

func MakeUnauthorizedErr(user string) error {
	return errUnauthorized(user)
}

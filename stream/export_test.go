package stream

func MakeMissingErr(id []byte) error {
	return errMissing(id)
}

func MakeExistsErr(id []byte) error {
	return errExists(id)
}

func MakeUnauthorizedErr(user string) error {
	return errUnauthorized(user)
}

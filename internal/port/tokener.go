package port

type TokenIssuer interface {
	Issue(userID string) (string, error)
}

type TokenVerifier interface {
	Verify(token string) (userID string, err error)
}

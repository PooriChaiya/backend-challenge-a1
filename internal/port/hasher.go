package port

type PasswordHasher interface {
	Hash(plain string) (string, error)
	Compare(hash, plain string) error // nil = match
}

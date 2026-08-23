package bcryptpw

import "golang.org/x/crypto/bcrypt"

type Hasher struct{ Cost int }

func New(cost int) Hasher {
	if cost == 0 {
		cost = bcrypt.DefaultCost
	}
	return Hasher{Cost: cost}
}

func (h Hasher) Hash(plain string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plain), h.Cost)
	return string(b), err
}

func (h Hasher) Compare(hash, plain string) error {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(plain))
}

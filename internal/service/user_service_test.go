package service

import (
	"context"
	"errors"
	"sync"
	"testing"

	"github.com/PooriChaiya/backend-challenge-a1/internal/domain"
)

// --- hand-written fakes ---

type memRepo struct {
	mu sync.Mutex
	m  map[string]domain.User
}

func newMemRepo() *memRepo { return &memRepo{m: map[string]domain.User{}} }

func (r *memRepo) Create(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, x := range r.m {
		if x.Email == u.Email {
			return domain.ErrDuplicateEmail
		}
	}
	r.m[u.ID] = *u
	return nil
}
func (r *memRepo) GetByID(_ context.Context, id string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	u, ok := r.m[id]
	if !ok {
		return nil, domain.ErrNotFound
	}
	return &u, nil
}
func (r *memRepo) GetByEmail(_ context.Context, email string) (*domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, u := range r.m {
		if u.Email == email {
			return &u, nil
		}
	}
	return nil, domain.ErrNotFound
}
func (r *memRepo) List(_ context.Context) ([]domain.User, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	out := make([]domain.User, 0, len(r.m))
	for _, u := range r.m {
		out = append(out, u)
	}
	return out, nil
}
func (r *memRepo) Update(_ context.Context, u *domain.User) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[u.ID]; !ok {
		return domain.ErrNotFound
	}
	r.m[u.ID] = *u
	return nil
}
func (r *memRepo) Delete(_ context.Context, id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()
	if _, ok := r.m[id]; !ok {
		return domain.ErrNotFound
	}
	delete(r.m, id)
	return nil
}
func (r *memRepo) Count(_ context.Context) (int64, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	return int64(len(r.m)), nil
}

type plainHasher struct{}

func (plainHasher) Hash(p string) (string, error) { return "H:" + p, nil }
func (plainHasher) Compare(hash, p string) error {
	if hash == "H:"+p {
		return nil
	}
	return errors.New("mismatch")
}

type stubIssuer struct{}

func (stubIssuer) Issue(id string) (string, error) { return "TOK:" + id, nil }

// --- tests ---

func newSvc() *UserService {
	return New(newMemRepo(), plainHasher{}, stubIssuer{})
}

func TestRegister_OK(t *testing.T) {
	s := newSvc()
	u, err := s.Register(context.Background(), RegisterInput{Name: "Alice", Email: "A@b.C", Password: "secret1"})
	if err != nil {
		t.Fatal(err)
	}
	if u.Email != "a@b.c" {
		t.Errorf("email not lowercased: %q", u.Email)
	}
	if u.PasswordHash == "secret1" || u.PasswordHash == "" {
		t.Errorf("password not hashed: %q", u.PasswordHash)
	}
	if u.ID == "" || u.CreatedAt.IsZero() {
		t.Errorf("missing fields: %+v", u)
	}
}

func TestRegister_BadInput(t *testing.T) {
	s := newSvc()
	cases := []RegisterInput{
		{Name: "", Email: "a@b.c", Password: "secret1"},
		{Name: "Alice", Email: "not-email", Password: "secret1"},
		{Name: "Alice", Email: "a@b.c", Password: "short"},
	}
	for i, in := range cases {
		if _, err := s.Register(context.Background(), in); !errors.Is(err, domain.ErrInvalidInput) {
			t.Errorf("case %d: want ErrInvalidInput, got %v", i, err)
		}
	}
}

func TestRegister_Duplicate(t *testing.T) {
	s := newSvc()
	in := RegisterInput{Name: "A", Email: "a@b.c", Password: "secret1"}
	if _, err := s.Register(context.Background(), in); err != nil {
		t.Fatal(err)
	}
	if _, err := s.Register(context.Background(), in); !errors.Is(err, domain.ErrDuplicateEmail) {
		t.Errorf("want ErrDuplicateEmail, got %v", err)
	}
}

func TestLogin(t *testing.T) {
	s := newSvc()
	u, _ := s.Register(context.Background(), RegisterInput{Name: "A", Email: "a@b.c", Password: "secret1"})

	tok, err := s.Login(context.Background(), "A@b.c", "secret1")
	if err != nil || tok != "TOK:"+u.ID {
		t.Errorf("login ok: got tok=%q err=%v", tok, err)
	}

	if _, err := s.Login(context.Background(), "a@b.c", "wrong"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("wrong pw: want ErrInvalidCredentials, got %v", err)
	}
	if _, err := s.Login(context.Background(), "nope@b.c", "secret1"); !errors.Is(err, domain.ErrInvalidCredentials) {
		t.Errorf("unknown email: want ErrInvalidCredentials, got %v", err)
	}
}

func TestGetUpdateDeleteCount(t *testing.T) {
	s := newSvc()
	u, _ := s.Register(context.Background(), RegisterInput{Name: "A", Email: "a@b.c", Password: "secret1"})

	got, err := s.Get(context.Background(), u.ID)
	if err != nil || got.ID != u.ID {
		t.Fatalf("get: %v %+v", err, got)
	}

	if _, err := s.Get(context.Background(), "missing"); !errors.Is(err, domain.ErrNotFound) {
		t.Errorf("get missing: want ErrNotFound, got %v", err)
	}

	newName := "Alice2"
	newEmail := "alice2@example.com"
	up, err := s.Update(context.Background(), u.ID, UpdateInput{Name: &newName, Email: &newEmail})
	if err != nil || up.Name != newName || up.Email != newEmail {
		t.Fatalf("update: %v %+v", err, up)
	}

	n, _ := s.Count(context.Background())
	if n != 1 {
		t.Errorf("count=1 want, got %d", n)
	}

	if err := s.Delete(context.Background(), u.ID); err != nil {
		t.Fatal(err)
	}
	n, _ = s.Count(context.Background())
	if n != 0 {
		t.Errorf("count=0 want, got %d", n)
	}
}

package httpapi

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/PooriChaiya/backend-challenge-a1/internal/domain"
	"github.com/PooriChaiya/backend-challenge-a1/internal/service"
)

// --- fakes (copied slim from service tests; ponytail: duplication beats a
// shared testutil package for two files) ---

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

type stubTokens struct{}

func (stubTokens) Issue(id string) (string, error) { return "TOK:" + id, nil }
func (stubTokens) Verify(t string) (string, error) {
	if !strings.HasPrefix(t, "TOK:") {
		return "", errors.New("bad")
	}
	return strings.TrimPrefix(t, "TOK:"), nil
}

// --- helpers ---

func newRouter() http.Handler {
	svc := service.New(newMemRepo(), plainHasher{}, stubTokens{})
	return NewRouter(NewHandlers(svc), stubTokens{})
}

func do(t *testing.T, r http.Handler, method, path, token string, body any) *httptest.ResponseRecorder {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		_ = json.NewEncoder(&buf).Encode(body)
	}
	req := httptest.NewRequest(method, path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

// --- tests ---

func TestRegisterAndListShape(t *testing.T) {
	r := newRouter()

	rr := do(t, r, "POST", "/register", "", map[string]string{"name": "Alice", "email": "a@b.c", "password": "secret1"})
	if rr.Code != http.StatusCreated {
		t.Fatalf("register status=%d body=%s", rr.Code, rr.Body.String())
	}
	// no password field ever
	if strings.Contains(strings.ToLower(rr.Body.String()), "password") {
		t.Errorf("register response leaks password: %s", rr.Body.String())
	}

	// duplicate → 409
	rr = do(t, r, "POST", "/register", "", map[string]string{"name": "A", "email": "a@b.c", "password": "secret1"})
	if rr.Code != http.StatusConflict {
		t.Errorf("dup want 409 got %d", rr.Code)
	}

	// login → token
	rr = do(t, r, "POST", "/login", "", map[string]string{"email": "a@b.c", "password": "secret1"})
	if rr.Code != http.StatusOK {
		t.Fatalf("login status=%d", rr.Code)
	}
	var lb map[string]string
	_ = json.Unmarshal(rr.Body.Bytes(), &lb)
	tok := lb["token"]
	if tok == "" {
		t.Fatal("no token returned")
	}

	// protected without token → 401
	if rr = do(t, r, "GET", "/users", "", nil); rr.Code != http.StatusUnauthorized {
		t.Errorf("no-token want 401 got %d", rr.Code)
	}
	// protected with bad token → 401
	if rr = do(t, r, "GET", "/users", "garbage", nil); rr.Code != http.StatusUnauthorized {
		t.Errorf("bad-token want 401 got %d", rr.Code)
	}

	// list → 200, one user, no password field
	rr = do(t, r, "GET", "/users", tok, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("list status=%d", rr.Code)
	}
	if strings.Contains(strings.ToLower(rr.Body.String()), "password") {
		t.Errorf("list response leaks password: %s", rr.Body.String())
	}
	var list []map[string]any
	_ = json.Unmarshal(rr.Body.Bytes(), &list)
	if len(list) != 1 {
		t.Fatalf("want 1 user, got %d", len(list))
	}
	id, _ := list[0]["id"].(string)
	if id == "" {
		t.Fatal("missing id in list response")
	}

	// update → 200
	rr = do(t, r, "PUT", "/users/"+id, tok, map[string]string{"name": "Alice2"})
	if rr.Code != http.StatusOK {
		t.Errorf("update status=%d body=%s", rr.Code, rr.Body.String())
	}

	// get missing → 404
	rr = do(t, r, "GET", "/users/nonexistent", tok, nil)
	if rr.Code != http.StatusNotFound {
		t.Errorf("missing want 404 got %d", rr.Code)
	}

	// delete → 204
	rr = do(t, r, "DELETE", "/users/"+id, tok, nil)
	if rr.Code != http.StatusNoContent {
		t.Errorf("delete want 204 got %d", rr.Code)
	}
}

func TestLoginWrongPassword(t *testing.T) {
	r := newRouter()
	_ = do(t, r, "POST", "/register", "", map[string]string{"name": "A", "email": "a@b.c", "password": "secret1"})
	rr := do(t, r, "POST", "/login", "", map[string]string{"email": "a@b.c", "password": "nope"})
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("want 401 got %d", rr.Code)
	}
}

func TestHealth(t *testing.T) {
	rr := do(t, newRouter(), "GET", "/healthz", "", nil)
	if rr.Code != http.StatusOK {
		t.Errorf("health want 200 got %d", rr.Code)
	}
}

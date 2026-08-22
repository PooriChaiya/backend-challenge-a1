// Package service is the application core — it orchestrates the ports
// (repository, hasher, token issuer) to fulfil use cases. It knows nothing
// about HTTP, Mongo, or JWT specifics.
package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"net/mail"
	"strings"
	"time"

	"github.com/PooriChaiya/backend-challenge-a1/internal/domain"
	"github.com/PooriChaiya/backend-challenge-a1/internal/port"
)

// randID returns a 16-byte hex id. ponytail: stdlib beats pulling in uuid.
func randID() string {
	var b [16]byte
	_, _ = rand.Read(b[:])
	return hex.EncodeToString(b[:])
}

type UserService struct {
	repo   port.UserRepository
	hasher port.PasswordHasher
	tokens port.TokenIssuer
	now    func() time.Time // ponytail: injectable for tests; time.Now in prod
	newID  func() string    // ponytail: injectable for tests; uuid in prod
}

func New(repo port.UserRepository, hasher port.PasswordHasher, tokens port.TokenIssuer) *UserService {
	return &UserService{
		repo:   repo,
		hasher: hasher,
		tokens: tokens,
		now:    time.Now,
		newID:  randID,
	}
}

type RegisterInput struct {
	Name, Email, Password string
}

func (s *UserService) Register(ctx context.Context, in RegisterInput) (*domain.User, error) {
	name := strings.TrimSpace(in.Name)
	email := strings.ToLower(strings.TrimSpace(in.Email))
	if name == "" || len(in.Password) < 6 {
		return nil, domain.ErrInvalidInput
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return nil, domain.ErrInvalidInput
	}
	hash, err := s.hasher.Hash(in.Password)
	if err != nil {
		return nil, err
	}
	u := &domain.User{
		ID:           s.newID(),
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		CreatedAt:    s.now().UTC(),
	}
	if err := s.repo.Create(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) Login(ctx context.Context, email, password string) (string, error) {
	u, err := s.repo.GetByEmail(ctx, strings.ToLower(strings.TrimSpace(email)))
	if err != nil {
		if errors.Is(err, domain.ErrNotFound) {
			return "", domain.ErrInvalidCredentials
		}
		return "", err
	}
	if err := s.hasher.Compare(u.PasswordHash, password); err != nil {
		return "", domain.ErrInvalidCredentials
	}
	return s.tokens.Issue(u.ID)
}

func (s *UserService) Get(ctx context.Context, id string) (*domain.User, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *UserService) List(ctx context.Context) ([]domain.User, error) {
	return s.repo.List(ctx)
}

type UpdateInput struct {
	Name, Email *string // nil = leave unchanged
}

func (s *UserService) Update(ctx context.Context, id string, in UpdateInput) (*domain.User, error) {
	u, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if in.Name != nil {
		n := strings.TrimSpace(*in.Name)
		if n == "" {
			return nil, domain.ErrInvalidInput
		}
		u.Name = n
	}
	if in.Email != nil {
		e := strings.ToLower(strings.TrimSpace(*in.Email))
		if _, err := mail.ParseAddress(e); err != nil {
			return nil, domain.ErrInvalidInput
		}
		u.Email = e
	}
	if err := s.repo.Update(ctx, u); err != nil {
		return nil, err
	}
	return u, nil
}

func (s *UserService) Delete(ctx context.Context, id string) error {
	return s.repo.Delete(ctx, id)
}

func (s *UserService) Count(ctx context.Context) (int64, error) {
	return s.repo.Count(ctx)
}

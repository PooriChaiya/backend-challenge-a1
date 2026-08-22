// Package port defines the interfaces the application service depends on
// (driven ports). Adapters in internal/adapter implement these.
package port

import (
	"context"

	"github.com/PooriChaiya/backend-challenge-a1/internal/domain"
)

type UserRepository interface {
	Create(ctx context.Context, u *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
	List(ctx context.Context) ([]domain.User, error)
	Update(ctx context.Context, u *domain.User) error
	Delete(ctx context.Context, id string) error
	Count(ctx context.Context) (int64, error)
}

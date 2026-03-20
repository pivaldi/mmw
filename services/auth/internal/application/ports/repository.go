package ports

import (
	"context"

	"github.com/google/uuid"
	authdomain "github.com/pivaldi/mmw/auth/internal/domain/auth"
	"github.com/pivaldi/mmw/auth/internal/domain/auth/user"
)

// UserRepository defines persistence operations for the User aggregate.
type UserRepository interface {
	Save(ctx context.Context, u *user.User) error
	FindByLogin(ctx context.Context, login user.Login) (*user.User, error)
	FindByID(ctx context.Context, id uuid.UUID) (*user.User, error)
	Update(ctx context.Context, u *user.User) error
	Delete(ctx context.Context, id uuid.UUID) error
}

// SessionRepository defines persistence operations for sessions.
type SessionRepository interface {
	Save(ctx context.Context, s *authdomain.Session) error
	FindByToken(ctx context.Context, token string) (*authdomain.Session, error)
}

// UnitOfWork manages transaction boundaries.
type UnitOfWork interface {
	WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error
}

// EventDispatcher writes domain events to the outbox table.
// It is called inside a transaction so events are written atomically
// with the domain record that emitted them.
type EventDispatcher interface {
	Dispatch(ctx context.Context, events []user.DomainEvent) error
}

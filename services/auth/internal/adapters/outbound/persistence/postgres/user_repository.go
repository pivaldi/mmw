package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	oglpguow "github.com/ovya/ogl/pg/uow"
	"github.com/pivaldi/mmw/auth/internal/domain/auth/user"
	"github.com/rotisserie/eris"
)

// UserRepository is the PostgreSQL implementation of ports.UserRepository.
type UserRepository struct {
	pool *pgxpool.Pool
}

func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

func (r *UserRepository) Save(ctx context.Context, u *user.User) error {
	exec := oglpguow.GetExecutor(ctx, r.pool)
	_, err := exec.Exec(ctx,
		`INSERT INTO auth.users (id, login, password_hash, created_at, updated_at)
		 VALUES ($1, $2, $3, $4, $5)`,
		u.ID(), u.Login().String(), u.PasswordHash().String(), u.CreatedAt(), u.UpdatedAt(),
	)

	return eris.Wrap(err, "save user")
}

func (r *UserRepository) FindByLogin(ctx context.Context, login user.Login) (*user.User, error) {
	exec := oglpguow.GetExecutor(ctx, r.pool)
	row := exec.QueryRow(ctx,
		`SELECT id, login, password_hash, created_at, updated_at FROM auth.users WHERE login = $1`,
		login.String(),
	)

	return scanUser(row)
}

func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (*user.User, error) {
	exec := oglpguow.GetExecutor(ctx, r.pool)
	row := exec.QueryRow(ctx,
		`SELECT id, login, password_hash, created_at, updated_at FROM auth.users WHERE id = $1`,
		id,
	)

	return scanUser(row)
}

func (r *UserRepository) Update(ctx context.Context, u *user.User) error {
	exec := oglpguow.GetExecutor(ctx, r.pool)
	_, err := exec.Exec(ctx,
		`UPDATE auth.users SET login = $1, password_hash = $2, updated_at = $3 WHERE id = $4`,
		u.Login().String(), u.PasswordHash().String(), u.UpdatedAt(), u.ID(),
	)

	return eris.Wrap(err, "update user")
}

func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	exec := oglpguow.GetExecutor(ctx, r.pool)
	_, err := exec.Exec(ctx, `DELETE FROM auth.users WHERE id = $1`, id)

	return eris.Wrap(err, "delete user")
}

// scanUser reconstructs a User aggregate from a pgx row.
// Uses ReconstructUser — never calls Create — so the stored password hash is preserved.
func scanUser(row pgx.Row) (*user.User, error) {
	var (
		id        uuid.UUID
		loginStr  string
		hashStr   string
		createdAt time.Time
		updatedAt time.Time
	)
	if err := row.Scan(&id, &loginStr, &hashStr, &createdAt, &updatedAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("user not found")
		}

		return nil, eris.Wrap(err, "scan user row")
	}

	login, err := user.NewLogin(loginStr)
	if err != nil {
		return nil, eris.Wrap(err, "reconstruct login")
	}
	ph := user.NewHashedPassword(hashStr)

	return user.ReconstructUser(id, login, ph, createdAt, updatedAt), nil
}

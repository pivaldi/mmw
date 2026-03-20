package postgres

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	oglpguow "github.com/ovya/ogl/pg/uow"
	authdomain "github.com/pivaldi/mmw/auth/internal/domain/auth"
	"github.com/rotisserie/eris"
)

// SessionRepository is the PostgreSQL implementation of ports.SessionRepository.
type SessionRepository struct {
	pool *pgxpool.Pool
}

func NewSessionRepository(pool *pgxpool.Pool) *SessionRepository {
	return &SessionRepository{pool: pool}
}

func (r *SessionRepository) Save(ctx context.Context, s *authdomain.Session) error {
	exec := oglpguow.GetExecutor(ctx, r.pool)
	_, err := exec.Exec(ctx,
		`INSERT INTO auth.sessions (id, user_id, token, expires_at) VALUES ($1, $2, $3, $4)`,
		s.ID(), s.UserID(), s.Token(), s.ExpiresAt(),
	)

	return eris.Wrap(err, "save session")
}

// FindByToken retrieves an unexpired session by its token.
func (r *SessionRepository) FindByToken(ctx context.Context, token string) (*authdomain.Session, error) {
	exec := oglpguow.GetExecutor(ctx, r.pool)
	row := exec.QueryRow(ctx,
		`SELECT id, user_id, token, expires_at FROM auth.sessions WHERE token = $1 AND expires_at > NOW()`,
		token,
	)

	var (
		id        uuid.UUID
		userID    uuid.UUID
		tok       string
		expiresAt time.Time
	)
	if err := row.Scan(&id, &userID, &tok, &expiresAt); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, errors.New("session not found or expired")
		}

		return nil, eris.Wrap(err, "scan session row")
	}

	return authdomain.ReconstructSession(id, userID, tok, expiresAt), nil
}

package auth

import (
	"time"

	"github.com/google/uuid"
)

// Session represents an authenticated user session.
type Session struct {
	id        uuid.UUID
	userID    uuid.UUID
	token     string
	expiresAt time.Time
}

// NewSession creates a new session valid for the given duration.
func NewSession(userID uuid.UUID, token string, duration time.Duration) *Session {
	return &Session{
		id:        uuid.New(),
		userID:    userID,
		token:     token,
		expiresAt: time.Now().Add(duration),
	}
}

// ReconstructSession restores a session from persisted state.
func ReconstructSession(id, userID uuid.UUID, token string, expiresAt time.Time) *Session {
	return &Session{id: id, userID: userID, token: token, expiresAt: expiresAt}
}

func (s *Session) ID() uuid.UUID        { return s.id }
func (s *Session) UserID() uuid.UUID    { return s.userID }
func (s *Session) Token() string        { return s.token }
func (s *Session) ExpiresAt() time.Time { return s.expiresAt }

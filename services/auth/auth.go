package auth

import (
	"context"

	"github.com/google/uuid"
	"github.com/rotisserie/eris"
)

type User struct {
	UUID         uuid.UUID
	Login        string
	PasswordHash string
}

type Service struct {
	// TODO: implement in the database.
	// db *pgxpool.Pool
}

// GetUserByID is Auth's standard internal method
func (s *Service) GetUserByID(_ context.Context, id string) (*User, error) {
	// SELECT id, email, password_hash FROM auth.users WHERE id = $1
	uid, err := uuid.Parse("4d698c6b-5532-4598-a7d7-db5e0c768ce6")
	if err != nil {
		return nil, eris.Wrap(err, "parsing uuid failed")
	}

	return &User{UUID: uid, Login: "admin"}, nil
}

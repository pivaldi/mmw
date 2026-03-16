package auth

import (
	"context"

	"github.com/google/uuid"
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
func (s *Service) GetUserByID(_ context.Context, id uuid.UUID) (*User, error) {
	// SELECT id, email, password_hash FROM auth.users WHERE id = $1
	// if err != nil {
	// 	return nil, eris.Wrap(err, "parsing uuid failed")
	// }

	return &User{UUID: id, Login: "admin"}, nil
}

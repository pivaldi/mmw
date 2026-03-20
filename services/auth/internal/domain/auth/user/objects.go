package user

import (
	"errors"
	"fmt"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

const maxLoginLen = 200

// Login is a validated user login value object.
type Login struct {
	value string
}

// NewLogin validates and wraps a login string.
func NewLogin(s string) (Login, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return Login{}, errors.New("login cannot be empty")
	}
	if len(s) > maxLoginLen {
		return Login{}, fmt.Errorf("login cannot exceed %d characters", maxLoginLen)
	}

	return Login{value: s}, nil
}

func (l Login) String() string { return l.value }

// PasswordHash is a bcrypt-hashed password value object.
type PasswordHash struct {
	hash string
}

// NewPasswordHash hashes a plaintext password with bcrypt.
func NewPasswordHash(plaintext string) (PasswordHash, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(plaintext), bcrypt.DefaultCost)
	if err != nil {
		return PasswordHash{}, fmt.Errorf("failed to generated password: %w", err)
	}

	return PasswordHash{hash: string(b)}, nil
}

// NewHashedPassword wraps an already-hashed string (used by repositories on reconstruction).
func NewHashedPassword(hash string) PasswordHash {
	return PasswordHash{hash: hash}
}

// Verify returns true if plaintext matches the stored hash.
func (p PasswordHash) Verify(plaintext string) bool {
	return bcrypt.CompareHashAndPassword([]byte(p.hash), []byte(plaintext)) == nil
}

// String returns the raw bcrypt hash string for persistence.
func (p PasswordHash) String() string { return p.hash }

package defauth

import (
	"context"
	"fmt"

	"github.com/google/uuid"
)

// DTOs. TODO; should be protobuf generated types
type UserDTO struct {
	UUID  uuid.UUID
	Login string
}

// The Public Interface of the AuthService
type AuthService interface {
	GetUser(ctx context.Context, id string) (*UserDTO, error)
	// ValidateToken verifies the JWT and checks the session exists in the DB.
	// Returns the userID on success, or an error if invalid/expired.
	ValidateToken(ctx context.Context, token string) (uuid.UUID, error)
}

// compile-time assertion
var _ AuthService = (*InprocClient)(nil)

// The InprocClient, a thin wrapper that accepts ANY implementation
type InprocClient struct {
	server AuthService
}

func NewInprocClient(server AuthService) *InprocClient {
	return &InprocClient{server: server}
}

func (c *InprocClient) GetUser(ctx context.Context, id string) (*UserDTO, error) {
	u, err := c.server.GetUser(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("%w", err)
	}

	return u, nil
}

func (c *InprocClient) ValidateToken(ctx context.Context, token string) (uuid.UUID, error) {
	id, err := c.server.ValidateToken(ctx, token)
	if err != nil {
		return uuid.Nil, fmt.Errorf("%w", err)
	}

	return id, nil
}

// ErrAuthUnavailable is returned by NoopAuthService methods.
var ErrAuthUnavailable = fmt.Errorf("auth service unavailable")

// NoopAuthService is a no-op implementation of AuthService that rejects every
// request. Use it in standalone deployments where auth is not wired.
type NoopAuthService struct{}

// compile-time assertion
var _ AuthService = (*NoopAuthService)(nil)

func (NoopAuthService) GetUser(_ context.Context, _ string) (*UserDTO, error) {
	return nil, ErrAuthUnavailable
}

func (NoopAuthService) ValidateToken(_ context.Context, _ string) (uuid.UUID, error) {
	return uuid.Nil, ErrAuthUnavailable
}

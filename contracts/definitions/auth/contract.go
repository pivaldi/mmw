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
}

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

package defauth

import (
	"context"
	"fmt"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	authv1 "github.com/pivaldi/mmw/contracts/gen/go/auth/v1"
	"github.com/pivaldi/mmw/contracts/gen/go/auth/v1/authv1connect"
)

// HttpClient adapts authv1connect.AuthServiceClient to defauth.AuthService.
// Use it when the auth service runs in a separate process and must be reached
// over HTTP (Connect protocol) rather than via an in-process call.
type HttpClient struct {
	client authv1connect.AuthServiceClient
}

// compile-time assertion
var _ AuthService = (*HttpClient)(nil)

// NewHttpClient wraps an authv1connect.AuthServiceClient so it satisfies
// defauth.AuthService. Construct the underlying client with:
//
//	authv1connect.NewAuthServiceClient(&http.Client{}, "http://localhost:8091")
func NewHttpClient(client authv1connect.AuthServiceClient) *HttpClient {
	return &HttpClient{client: client}
}

// ValidateToken calls the auth service's ValidateToken RPC and returns the
// userID on success, or an error if the token is invalid or the call fails.
func (c *HttpClient) ValidateToken(ctx context.Context, token string) (uuid.UUID, error) {
	resp, err := c.client.ValidateToken(ctx, connect.NewRequest(&authv1.ValidateTokenRequest{Token: token}))
	if err != nil {
		return uuid.Nil, fmt.Errorf("validating token: %w", err)
	}

	if !resp.Msg.IsValid {
		return uuid.Nil, fmt.Errorf("token is invalid or expired")
	}

	id, err := uuid.Parse(resp.Msg.UserId)
	if err != nil {
		return uuid.Nil, fmt.Errorf("parsing user ID from auth response: %w", err)
	}

	return id, nil
}

// GetUser is not yet exposed as an RPC in the auth service proto.
// It returns ErrAuthUnavailable when called through the HTTP client.
func (c *HttpClient) GetUser(_ context.Context, _ string) (*UserDTO, error) {
	return nil, ErrAuthUnavailable
}

package connect

import (
	"context"
	"errors"

	"connectrpc.com/connect"
	"github.com/google/uuid"
	"github.com/pivaldi/mmw/auth/internal/application"
	authv1 "github.com/pivaldi/mmw/contracts/gen/go/auth/v1"
	"github.com/pivaldi/mmw/contracts/gen/go/auth/v1/authv1connect"
)

// AuthHandler is the Connect RPC handler for the auth service.
type AuthHandler struct {
	svc *application.AuthApplicationService
}

// NewAuthHandler creates a new AuthHandler.
func NewAuthHandler(svc *application.AuthApplicationService) authv1connect.AuthServiceHandler {
	return &AuthHandler{svc: svc}
}

func (h *AuthHandler) Register(
	ctx context.Context,
	req *connect.Request[authv1.RegisterRequest],
) (*connect.Response[authv1.RegisterResponse], error) {
	userID, err := h.svc.Register(ctx, req.Msg.Login, req.Msg.Password)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, err)
	}

	return connect.NewResponse(&authv1.RegisterResponse{UserId: userID.String()}), nil
}

func (h *AuthHandler) Login(
	ctx context.Context,
	req *connect.Request[authv1.LoginRequest],
) (*connect.Response[authv1.LoginResponse], error) {
	token, userID, err := h.svc.Login(ctx, req.Msg.Login, req.Msg.Password)
	if err != nil {
		if errors.Is(err, application.ErrInvalidCredentials) {
			return nil, connect.NewError(connect.CodeUnauthenticated, err)
		}

		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&authv1.LoginResponse{
		Token:  token,
		UserId: userID.String(),
	}), nil
}

func (h *AuthHandler) ValidateToken(
	ctx context.Context,
	req *connect.Request[authv1.ValidateTokenRequest],
) (*connect.Response[authv1.ValidateTokenResponse], error) {
	userID, err := h.svc.ValidateToken(ctx, req.Msg.Token)
	if err != nil {
		return connect.NewResponse(&authv1.ValidateTokenResponse{IsValid: false}), nil
	}

	return connect.NewResponse(&authv1.ValidateTokenResponse{
		UserId:  userID.String(),
		IsValid: true,
	}), nil
}

func (h *AuthHandler) ChangePassword(
	ctx context.Context,
	req *connect.Request[authv1.ChangePasswordRequest],
) (*connect.Response[authv1.ChangePasswordResponse], error) {
	userID, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
	}

	if err := h.svc.ChangePassword(ctx, userID, req.Msg.OldPassword, req.Msg.NewPassword); err != nil {
		if errors.Is(err, application.ErrUserNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}
		if errors.Is(err, application.ErrInvalidCredentials) {
			return nil, connect.NewError(connect.CodeInvalidArgument, err)
		}

		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&authv1.ChangePasswordResponse{}), nil
}

func (h *AuthHandler) DeleteUser(
	ctx context.Context,
	req *connect.Request[authv1.DeleteUserRequest],
) (*connect.Response[authv1.DeleteUserResponse], error) {
	userID, err := uuid.Parse(req.Msg.UserId)
	if err != nil {
		return nil, connect.NewError(connect.CodeInvalidArgument, errors.New("invalid user_id"))
	}

	if err := h.svc.DeleteUser(ctx, userID); err != nil {
		if errors.Is(err, application.ErrUserNotFound) {
			return nil, connect.NewError(connect.CodeNotFound, err)
		}

		return nil, connect.NewError(connect.CodeInternal, err)
	}

	return connect.NewResponse(&authv1.DeleteUserResponse{}), nil
}

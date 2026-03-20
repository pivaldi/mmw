package application_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	authdomain "github.com/pivaldi/mmw/auth/internal/domain/auth"
	"github.com/pivaldi/mmw/auth/internal/domain/auth/user"
	"github.com/pivaldi/mmw/auth/internal/application"
	"github.com/pivaldi/mmw/auth/internal/application/ports"
)

// --- Mock repos ---

type mockUserRepo struct {
	saved     *user.User
	byLogin   map[string]*user.User
	byID      map[uuid.UUID]*user.User
	updated   *user.User
	deletedID uuid.UUID
}

func (m *mockUserRepo) Save(_ context.Context, u *user.User) error {
	m.saved = u
	if m.byLogin == nil {
		m.byLogin = make(map[string]*user.User)
	}
	if m.byID == nil {
		m.byID = make(map[uuid.UUID]*user.User)
	}
	m.byLogin[u.Login().String()] = u
	m.byID[u.ID()] = u
	return nil
}
func (m *mockUserRepo) FindByLogin(_ context.Context, login user.Login) (*user.User, error) {
	u, ok := m.byLogin[login.String()]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}
func (m *mockUserRepo) FindByID(_ context.Context, id uuid.UUID) (*user.User, error) {
	u, ok := m.byID[id]
	if !ok {
		return nil, errors.New("not found")
	}
	return u, nil
}
func (m *mockUserRepo) Update(_ context.Context, u *user.User) error { m.updated = u; return nil }
func (m *mockUserRepo) Delete(_ context.Context, id uuid.UUID) error { m.deletedID = id; return nil }

type mockSessionRepo struct {
	saved   *authdomain.Session
	byToken map[string]*authdomain.Session
}

func (m *mockSessionRepo) Save(_ context.Context, s *authdomain.Session) error {
	m.saved = s
	if m.byToken == nil {
		m.byToken = make(map[string]*authdomain.Session)
	}
	m.byToken[s.Token()] = s
	return nil
}
func (m *mockSessionRepo) FindByToken(_ context.Context, token string) (*authdomain.Session, error) {
	s, ok := m.byToken[token]
	if !ok {
		return nil, errors.New("not found")
	}
	return s, nil
}

type mockDispatcher struct{ dispatched []user.DomainEvent }

func (m *mockDispatcher) Dispatch(_ context.Context, events []user.DomainEvent) error {
	m.dispatched = append(m.dispatched, events...)
	return nil
}

type mockUoW struct{}

func (m *mockUoW) WithTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

// Interface compliance checks
var _ ports.UserRepository    = (*mockUserRepo)(nil)
var _ ports.SessionRepository = (*mockSessionRepo)(nil)
var _ ports.UnitOfWork        = (*mockUoW)(nil)
var _ ports.EventDispatcher   = (*mockDispatcher)(nil)

func newTestService() (*application.AuthApplicationService, *mockUserRepo, *mockSessionRepo, *mockDispatcher) {
	ur := &mockUserRepo{}
	sr := &mockSessionRepo{}
	d := &mockDispatcher{}
	svc := application.NewAuthService(ur, sr, &mockUoW{}, d, "test-secret-key-32-bytes-minimum!!")
	return svc, ur, sr, d
}

func TestRegister_createsUser(t *testing.T) {
	svc, ur, _, d := newTestService()
	id, err := svc.Register(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("Register: %v", err)
	}
	if id == uuid.Nil {
		t.Error("expected non-nil user ID")
	}
	if ur.saved == nil {
		t.Error("expected user to be saved")
	}
	if len(d.dispatched) == 0 {
		t.Error("expected UserRegistered event to be dispatched")
	}
	if d.dispatched[0].EventType() != "auth.user.registered" {
		t.Errorf("expected auth.user.registered, got %s", d.dispatched[0].EventType())
	}
}

func TestLogin_returnsToken(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, _ = svc.Register(context.Background(), "alice", "password123")

	token, userID, err := svc.Login(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}
	if token == "" {
		t.Error("expected non-empty token")
	}
	if userID == uuid.Nil {
		t.Error("expected non-nil userID")
	}
}

func TestLogin_wrongPassword(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, _ = svc.Register(context.Background(), "alice", "password123")

	_, _, err := svc.Login(context.Background(), "alice", "wrongpassword")
	if err == nil {
		t.Fatal("expected error for wrong password")
	}
	if !errors.Is(err, application.ErrInvalidCredentials) {
		t.Errorf("expected ErrInvalidCredentials, got %v", err)
	}
}

func TestValidateToken_validToken(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, _ = svc.Register(context.Background(), "alice", "password123")
	token, _, err := svc.Login(context.Background(), "alice", "password123")
	if err != nil {
		t.Fatalf("Login: %v", err)
	}

	userID, err := svc.ValidateToken(context.Background(), token)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if userID == uuid.Nil {
		t.Error("expected non-nil userID")
	}
}

func TestValidateToken_invalidJWT(t *testing.T) {
	svc, _, _, _ := newTestService()
	_, err := svc.ValidateToken(context.Background(), "not-a-jwt")
	if err == nil {
		t.Fatal("expected error for invalid JWT")
	}
	if !errors.Is(err, application.ErrInvalidToken) {
		t.Errorf("expected ErrInvalidToken, got %v", err)
	}
}

func TestDeleteUser_callsRepoDelete(t *testing.T) {
	svc, ur, _, _ := newTestService()
	id, _ := svc.Register(context.Background(), "alice", "password123")

	err := svc.DeleteUser(context.Background(), id)
	if err != nil {
		t.Fatalf("DeleteUser: %v", err)
	}
	if ur.deletedID != id {
		t.Errorf("expected Delete called with %s, got %s", id, ur.deletedID)
	}
}

func TestChangePassword_updatesUser(t *testing.T) {
	svc, ur, _, _ := newTestService()
	id, _ := svc.Register(context.Background(), "alice", "oldpassword")

	err := svc.ChangePassword(context.Background(), id, "oldpassword", "newpassword")
	if err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	if ur.updated == nil {
		t.Error("expected Update to be called")
	}
}

func TestChangePassword_wrongOldPassword(t *testing.T) {
	svc, _, _, _ := newTestService()
	id, _ := svc.Register(context.Background(), "alice", "oldpassword")

	err := svc.ChangePassword(context.Background(), id, "wrongold", "newpassword")
	if err == nil {
		t.Fatal("expected error for wrong old password")
	}
}

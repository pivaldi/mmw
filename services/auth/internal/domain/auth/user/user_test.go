package user_test

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pivaldi/mmw/auth/internal/domain/auth/user"
)

func mustLogin(t *testing.T, s string) user.Login {
	t.Helper()
	l, err := user.NewLogin(s)
	if err != nil {
		t.Fatalf("NewLogin(%q): %v", s, err)
	}
	return l
}

func TestCreate_emitsUserRegistered(t *testing.T) {
	id := uuid.New()
	u, err := user.Create(id, mustLogin(t, "alice"), "password123")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	events := u.ClearEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event, got %d", len(events))
	}
	if events[0].EventType() != "auth.user.registered" {
		t.Errorf("expected auth.user.registered, got %s", events[0].EventType())
	}
	if events[0].AggregateID() != id.String() {
		t.Errorf("expected aggregateID %s, got %s", id.String(), events[0].AggregateID())
	}
}

func TestCreate_setsFieldsCorrectly(t *testing.T) {
	id := uuid.New()
	u, err := user.Create(id, mustLogin(t, "bob"), "secret")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if u.ID() != id {
		t.Errorf("expected ID %s, got %s", id, u.ID())
	}
	if u.Login().String() != "bob" {
		t.Errorf("expected login bob, got %s", u.Login())
	}
}

func TestCreate_emptyPasswordFails(t *testing.T) {
	_, err := user.Create(uuid.New(), mustLogin(t, "alice"), "")
	if err == nil {
		t.Fatal("expected error for empty password")
	}
}

func TestCheckPassword_correct(t *testing.T) {
	u, _ := user.Create(uuid.New(), mustLogin(t, "alice"), "mypassword")
	if !u.CheckPassword("mypassword") {
		t.Error("expected CheckPassword to return true")
	}
}

func TestCheckPassword_wrong(t *testing.T) {
	u, _ := user.Create(uuid.New(), mustLogin(t, "alice"), "mypassword")
	if u.CheckPassword("wrongpassword") {
		t.Error("expected CheckPassword to return false")
	}
}

func TestChangePassword_emitsPasswordChanged(t *testing.T) {
	u, _ := user.Create(uuid.New(), mustLogin(t, "alice"), "oldpass")
	u.ClearEvents()
	if err := u.ChangePassword("oldpass", "newpass"); err != nil {
		t.Fatalf("ChangePassword: %v", err)
	}
	events := u.ClearEvents()
	if len(events) != 1 || events[0].EventType() != "auth.user.password_changed" {
		t.Errorf("expected password_changed event, got %v", events)
	}
}

func TestChangePassword_wrongOldPassword(t *testing.T) {
	u, _ := user.Create(uuid.New(), mustLogin(t, "alice"), "oldpass")
	u.ClearEvents()
	if err := u.ChangePassword("wrongold", "newpass"); err == nil {
		t.Fatal("expected error for wrong old password")
	}
	if len(u.ClearEvents()) != 0 {
		t.Error("expected no events on failed password change")
	}
}

func TestChangePassword_emptyNewPasswordFails(t *testing.T) {
	u, _ := user.Create(uuid.New(), mustLogin(t, "alice"), "oldpass")
	u.ClearEvents()
	if err := u.ChangePassword("oldpass", ""); err == nil {
		t.Fatal("expected error for empty new password")
	}
	if len(u.ClearEvents()) != 0 {
		t.Error("expected no events on failed password change")
	}
}

func TestMarkLoggedIn_emitsUserLoggedIn(t *testing.T) {
	u, _ := user.Create(uuid.New(), mustLogin(t, "alice"), "pass")
	u.ClearEvents()
	u.MarkLoggedIn()
	events := u.ClearEvents()
	if len(events) != 1 || events[0].EventType() != "auth.user.logged_in" {
		t.Errorf("expected logged_in event, got %v", events)
	}
}

func TestDelete_emitsUserDeleted(t *testing.T) {
	u, _ := user.Create(uuid.New(), mustLogin(t, "alice"), "pass")
	u.ClearEvents()
	u.Delete()
	events := u.ClearEvents()
	if len(events) != 1 || events[0].EventType() != "auth.user.deleted" {
		t.Errorf("expected deleted event, got %v", events)
	}
}

func TestUserRegistered_MarshalJSON(t *testing.T) {
	id := uuid.New()
	u, _ := user.Create(id, mustLogin(t, "alice"), "password123")
	events := u.ClearEvents()
	if len(events) != 1 {
		t.Fatalf("expected 1 event")
	}

	b, err := json.Marshal(events[0])
	if err != nil {
		t.Fatalf("MarshalJSON: %v", err)
	}
	s := string(b)
	if !strings.Contains(s, `"aggregate_id"`) {
		t.Errorf("expected aggregate_id in JSON, got: %s", s)
	}
	if !strings.Contains(s, `"alice"`) {
		t.Errorf("expected login in JSON, got: %s", s)
	}
}

func TestReconstructUser_noEvents(t *testing.T) {
	id := uuid.New()
	l, _ := user.NewLogin("alice")
	ph, _ := user.NewPasswordHash("secret")
	now := time.Now()
	u := user.ReconstructUser(id, l, ph, now, now)
	if len(u.ClearEvents()) != 0 {
		t.Error("ReconstructUser must not emit events")
	}
	if !u.CheckPassword("secret") {
		t.Error("ReconstructUser must preserve password hash")
	}
}

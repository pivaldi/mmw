package user_test

import (
	"strings"
	"testing"

	"github.com/pivaldi/mmw/auth/internal/domain/auth/user"
)

func TestNewLogin_valid(t *testing.T) {
	l, err := user.NewLogin("alice")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if l.String() != "alice" {
		t.Errorf("expected alice, got %s", l.String())
	}
}

func TestNewLogin_empty(t *testing.T) {
	_, err := user.NewLogin("")
	if err == nil {
		t.Fatal("expected error for empty login")
	}
}

func TestNewLogin_tooLong(t *testing.T) {
	_, err := user.NewLogin(strings.Repeat("a", 201))
	if err == nil {
		t.Fatal("expected error for login > 200 chars")
	}
}

func TestNewPasswordHash_hashesInput(t *testing.T) {
	ph, err := user.NewPasswordHash("secret123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ph.String() == "secret123" {
		t.Error("expected hash, got plaintext")
	}
	if ph.String() == "" {
		t.Error("expected non-empty hash")
	}
}

func TestPasswordHash_Verify_correct(t *testing.T) {
	ph, _ := user.NewPasswordHash("secret123")
	if !ph.Verify("secret123") {
		t.Error("expected Verify to return true for correct password")
	}
}

func TestPasswordHash_Verify_wrong(t *testing.T) {
	ph, _ := user.NewPasswordHash("secret123")
	if ph.Verify("wrong") {
		t.Error("expected Verify to return false for wrong password")
	}
}

func TestNewHashedPassword_restoresHash(t *testing.T) {
	ph, _ := user.NewPasswordHash("secret123")
	restored := user.NewHashedPassword(ph.String())
	if !restored.Verify("secret123") {
		t.Error("restored hash should verify original password")
	}
}

package user

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

// User is the auth bounded context aggregate root.
type User struct {
	id           uuid.UUID
	login        Login
	passwordHash PasswordHash
	createdAt    time.Time
	updatedAt    time.Time
	events       []DomainEvent
}

// Create validates inputs, hashes the password, and emits UserRegistered.
func Create(id uuid.UUID, login Login, plainPassword string) (*User, error) {
	if plainPassword == "" {
		return nil, errors.New("password cannot be empty")
	}
	ph, err := NewPasswordHash(plainPassword)
	if err != nil {
		return nil, err
	}
	now := time.Now()
	u := &User{
		id:           id,
		login:        login,
		passwordHash: ph,
		createdAt:    now,
		updatedAt:    now,
	}
	u.addEvent(UserRegistered{
		aggregateID: id.String(),
		occurredAt:  now,
		Login:       login.String(),
	})

	return u, nil
}

// ReconstructUser restores a User from persisted state without re-hashing.
// Only repositories should call this.
func ReconstructUser(id uuid.UUID, login Login, passwordHash PasswordHash, createdAt, updatedAt time.Time) *User {
	return &User{
		id:           id,
		login:        login,
		passwordHash: passwordHash,
		createdAt:    createdAt,
		updatedAt:    updatedAt,
	}
}

func (u *User) ID() uuid.UUID              { return u.id }
func (u *User) Login() Login               { return u.login }
func (u *User) PasswordHash() PasswordHash { return u.passwordHash }
func (u *User) CreatedAt() time.Time       { return u.createdAt }
func (u *User) UpdatedAt() time.Time       { return u.updatedAt }

// CheckPassword returns true if plaintext matches the stored hash.
func (u *User) CheckPassword(plaintext string) bool {
	return u.passwordHash.Verify(plaintext)
}

// ChangePassword verifies the old password then replaces it with a new hash.
// Emits PasswordChanged on success.
func (u *User) ChangePassword(oldPassword, newPassword string) error {
	if !u.passwordHash.Verify(oldPassword) {
		return errors.New("invalid current password")
	}
	if newPassword == "" {
		return errors.New("password cannot be empty")
	}
	ph, err := NewPasswordHash(newPassword)
	if err != nil {
		return err
	}
	u.passwordHash = ph
	u.updatedAt = time.Now()
	u.addEvent(PasswordChanged{
		aggregateID: u.id.String(),
		occurredAt:  u.updatedAt,
	})

	return nil
}

// MarkLoggedIn emits UserLoggedIn without changing aggregate state.
func (u *User) MarkLoggedIn() {
	u.addEvent(UserLoggedIn{
		aggregateID: u.id.String(),
		occurredAt:  time.Now(),
	})
}

// Delete emits UserDeleted without changing aggregate state.
func (u *User) Delete() {
	u.addEvent(UserDeleted{
		aggregateID: u.id.String(),
		occurredAt:  time.Now(),
	})
}

// ClearEvents returns all pending events and clears the internal slice.
// Called by the outbox dispatcher after writing events to the DB.
func (u *User) ClearEvents() []DomainEvent {
	evts := u.events
	u.events = nil

	return evts
}

// addEvent adds a domain event to the unpublished events list
func (t *User) addEvent(event DomainEvent) {
	t.events = append(t.events, event)
}

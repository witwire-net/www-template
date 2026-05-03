package domain

import (
	"errors"
	"testing"
	"time"
)

// [UT-AUTH-BE-BND-001] NewPasskeyCredential の空ハンドル・無効 ID で適切なエラーが返ることをテストする。

func TestNewPasskeyCredential_InvalidHandle(t *testing.T) {
	t.Parallel()
	t.Run("empty credentialHandle returns ErrInvalidPasskeyCredential", func(t *testing.T) {
		t.Parallel()
		_, err := NewPasskeyCredential(
			"01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"01ARZ3NDEKTSV4RRFFQ69G5FAW",
			"user@example.com",
			"",
			time.Now(),
		)
		if !errors.Is(err, ErrInvalidPasskeyCredential) {
			t.Fatalf("expected ErrInvalidPasskeyCredential, got %v", err)
		}
	})

	t.Run("whitespace-only credentialHandle returns ErrInvalidPasskeyCredential", func(t *testing.T) {
		t.Parallel()
		_, err := NewPasskeyCredential(
			"01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"01ARZ3NDEKTSV4RRFFQ69G5FAW",
			"user@example.com",
			"   ",
			time.Now(),
		)
		if !errors.Is(err, ErrInvalidPasskeyCredential) {
			t.Fatalf("expected ErrInvalidPasskeyCredential, got %v", err)
		}
	})

	t.Run("invalid id returns ErrInvalidAuthID", func(t *testing.T) {
		t.Parallel()
		_, err := NewPasskeyCredential(
			"not-a-ulid",
			"01ARZ3NDEKTSV4RRFFQ69G5FAW",
			"user@example.com",
			"some-handle",
			time.Now(),
		)
		if !errors.Is(err, ErrInvalidAuthID) {
			t.Fatalf("expected ErrInvalidAuthID, got %v", err)
		}
	})

	t.Run("invalid accountID returns ErrInvalidAccountID", func(t *testing.T) {
		t.Parallel()
		_, err := NewPasskeyCredential(
			"01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"not-a-ulid",
			"user@example.com",
			"some-handle",
			time.Now(),
		)
		if !errors.Is(err, ErrInvalidAccountID) {
			t.Fatalf("expected ErrInvalidAccountID, got %v", err)
		}
	})

	t.Run("valid input succeeds", func(t *testing.T) {
		t.Parallel()
		now := time.Now().UTC()
		cred, err := NewPasskeyCredential(
			"01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"01ARZ3NDEKTSV4RRFFQ69G5FAW",
			"user@example.com",
			"some-handle",
			now,
		)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}
		if cred.ID() != "01ARZ3NDEKTSV4RRFFQ69G5FAV" {
			t.Errorf("unexpected id %q", cred.ID())
		}
		if cred.CredentialHandle() != "some-handle" {
			t.Errorf("unexpected handle %q", cred.CredentialHandle())
		}
		if !cred.CreatedAt().Equal(now) {
			t.Errorf("unexpected createdAt %v", cred.CreatedAt())
		}
	})
}

func TestAuthAccountBackwardCompatAccessors(t *testing.T) {
	t.Parallel()
	t.Run("PasskeyCredentialID returns first credential id", func(t *testing.T) {
		t.Parallel()
		account, err := NewAuthAccount(
			"01ARZ3NDEKTSV4RRFFQ69G5FAV",
			"user@example.com",
			"user@example.com",
			"01ARZ3NDEKTSV4RRFFQ69G5FB0",
			"my-handle",
		)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if account.PasskeyCredentialID() != "01ARZ3NDEKTSV4RRFFQ69G5FB0" {
			t.Errorf("unexpected credential id %q", account.PasskeyCredentialID())
		}
		if account.CredentialHandle() != "my-handle" {
			t.Errorf("unexpected handle %q", account.CredentialHandle())
		}
		if len(account.Credentials()) != 1 {
			t.Errorf("expected 1 credential, got %d", len(account.Credentials()))
		}
	})
}

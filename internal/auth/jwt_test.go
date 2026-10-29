package auth_test

import (
	"testing"
	"time"

	"github.com/amsoft/servicewatch/internal/auth"
)

var testUser = &auth.User{
	ID:    42,
	Email: "test@example.com",
	Name:  "Test User",
}

const testSecret = "super-secret-jwt-key-for-tests-minimum-32chars"

func TestIssueAndParseToken(t *testing.T) {
	token, expiresAt, err := auth.IssueToken(testUser, testSecret, time.Hour)
	if err != nil {
		t.Fatalf("IssueToken error: %v", err)
	}
	if token == "" {
		t.Fatal("expected non-empty token")
	}
	if expiresAt.Before(time.Now()) {
		t.Fatal("token already expired at issue time")
	}

	claims, err := auth.ParseToken(token, testSecret)
	if err != nil {
		t.Fatalf("ParseToken error: %v", err)
	}
	if claims.UserID != testUser.ID {
		t.Errorf("expected UserID %d, got %d", testUser.ID, claims.UserID)
	}
	if claims.Email != testUser.Email {
		t.Errorf("expected email %q, got %q", testUser.Email, claims.Email)
	}
}

func TestParseToken_WrongSecret(t *testing.T) {
	token, _, _ := auth.IssueToken(testUser, testSecret, time.Hour)
	_, err := auth.ParseToken(token, "wrong-secret-key-that-is-at-least-32-chars!")
	if err == nil {
		t.Fatal("expected error for wrong secret, got nil")
	}
}

func TestParseToken_Expired(t *testing.T) {
	token, _, _ := auth.IssueToken(testUser, testSecret, -time.Second) // already expired
	_, err := auth.ParseToken(token, testSecret)
	if err == nil {
		t.Fatal("expected error for expired token, got nil")
	}
}

func TestParseToken_Malformed(t *testing.T) {
	_, err := auth.ParseToken("not.a.jwt", testSecret)
	if err == nil {
		t.Fatal("expected error for malformed token, got nil")
	}
}

func TestParseToken_Empty(t *testing.T) {
	_, err := auth.ParseToken("", testSecret)
	if err == nil {
		t.Fatal("expected error for empty token, got nil")
	}
}

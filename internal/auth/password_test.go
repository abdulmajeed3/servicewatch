package auth_test

import (
	"testing"

	"github.com/amsoft/servicewatch/internal/auth"
)

func TestHashPassword_ValidPassword(t *testing.T) {
	hash, err := auth.HashPassword("ValidPass1", 4) // cost 4 for speed in tests
	if err != nil {
		t.Fatalf("expected no error, got: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == "ValidPass1" {
		t.Fatal("hash must not equal plaintext")
	}
}

func TestCheckPassword_Matches(t *testing.T) {
	hash, _ := auth.HashPassword("MyPassword1", 4)
	if err := auth.CheckPassword(hash, "MyPassword1"); err != nil {
		t.Fatalf("expected match, got: %v", err)
	}
}

func TestCheckPassword_Mismatch(t *testing.T) {
	hash, _ := auth.HashPassword("CorrectPass1", 4)
	err := auth.CheckPassword(hash, "WrongPass1")
	if err == nil {
		t.Fatal("expected mismatch error, got nil")
	}
}

func TestHashPassword_TooShort(t *testing.T) {
	_, err := auth.HashPassword("Ab1", 4)
	if err == nil {
		t.Fatal("expected error for short password")
	}
}

func TestHashPassword_TooLong(t *testing.T) {
	long := make([]byte, 129)
	for i := range long {
		long[i] = 'A'
	}
	long[0] = 'a'
	long[1] = '1'
	_, err := auth.HashPassword(string(long), 4)
	if err == nil {
		t.Fatal("expected error for password > 128 chars")
	}
}

func TestHashPassword_NoUppercase(t *testing.T) {
	_, err := auth.HashPassword("onlylower1", 4)
	if err == nil {
		t.Fatal("expected error for no uppercase")
	}
}

func TestHashPassword_NoDigit(t *testing.T) {
	_, err := auth.HashPassword("OnlyLetters", 4)
	if err == nil {
		t.Fatal("expected error for no digit")
	}
}

func TestHashPassword_BoundaryLength(t *testing.T) {
	// Exactly 8 chars with requirements met.
	_, err := auth.HashPassword("Abcdef1!", 4)
	if err != nil {
		t.Fatalf("expected valid 8-char password to pass, got: %v", err)
	}
}

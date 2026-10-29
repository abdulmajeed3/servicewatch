package auth

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/mail"
	"strings"
	"time"
)

// Service handles authentication business logic.
type Service struct {
	store       *Store
	jwtSecret   string
	tokenExpiry time.Duration
	bcryptCost  int
	log         *slog.Logger
}

func NewService(store *Store, jwtSecret string, tokenExpiry time.Duration, bcryptCost int, log *slog.Logger) *Service {
	return &Service{
		store:       store,
		jwtSecret:   jwtSecret,
		tokenExpiry: tokenExpiry,
		bcryptCost:  bcryptCost,
		log:         log,
	}
}

// Register creates a new user account.
func (s *Service) Register(ctx context.Context, req *RegisterRequest) (*TokenResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))
	req.Name = strings.TrimSpace(req.Name)

	if err := validateEmail(req.Email); err != nil {
		return nil, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return nil, errors.New("name is required")
	}
	if len(req.Name) > 100 {
		return nil, errors.New("name must be at most 100 characters")
	}

	hash, err := HashPassword(req.Password, s.bcryptCost)
	if err != nil {
		return nil, err
	}

	user, err := s.store.CreateUser(ctx, req.Email, req.Name, hash)
	if err != nil {
		return nil, err
	}

	return s.buildTokenResponse(user)
}

// Login authenticates a user and returns a token.
func (s *Service) Login(ctx context.Context, req *LoginRequest) (*TokenResponse, error) {
	req.Email = strings.ToLower(strings.TrimSpace(req.Email))

	user, err := s.store.GetUserByEmail(ctx, req.Email)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			// Return same error message to avoid email enumeration.
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user: %w", err)
	}

	if !user.IsActive {
		return nil, ErrInvalidCredentials
	}

	if err := CheckPassword(user.PasswordHash, req.Password); err != nil {
		return nil, err
	}

	return s.buildTokenResponse(user)
}

// GetUser returns the user with the given ID.
func (s *Service) GetUser(ctx context.Context, id int64) (*User, error) {
	return s.store.GetUserByID(ctx, id)
}

// UpdateProfile updates user profile fields.
func (s *Service) UpdateProfile(ctx context.Context, id int64, req *UpdateProfileRequest) (*User, error) {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return nil, errors.New("name is required")
	}
	if len(name) > 100 {
		return nil, errors.New("name must be at most 100 characters")
	}
	return s.store.UpdateUser(ctx, id, name)
}

// ChangePassword updates the user's password after verifying the current one.
func (s *Service) ChangePassword(ctx context.Context, id int64, req *ChangePasswordRequest) error {
	user, err := s.store.GetUserByID(ctx, id)
	if err != nil {
		return err
	}
	if err := CheckPassword(user.PasswordHash, req.CurrentPassword); err != nil {
		return ErrInvalidCredentials
	}
	hash, err := HashPassword(req.NewPassword, s.bcryptCost)
	if err != nil {
		return err
	}
	return s.store.UpdatePassword(ctx, id, hash)
}

// ValidateToken parses and validates a JWT string, returning its claims.
func (s *Service) ValidateToken(tokenStr string) (*Claims, error) {
	return ParseToken(tokenStr, s.jwtSecret)
}

func (s *Service) buildTokenResponse(user *User) (*TokenResponse, error) {
	token, expiresAt, err := IssueToken(user, s.jwtSecret, s.tokenExpiry)
	if err != nil {
		return nil, fmt.Errorf("issue token: %w", err)
	}
	u := *user
	u.PasswordHash = ""
	return &TokenResponse{
		Token:     token,
		ExpiresAt: expiresAt,
		User:      &u,
	}, nil
}

func validateEmail(email string) error {
	if email == "" {
		return errors.New("email is required")
	}
	if _, err := mail.ParseAddress(email); err != nil {
		return errors.New("email is invalid")
	}
	if len(email) > 254 {
		return errors.New("email is too long")
	}
	return nil
}

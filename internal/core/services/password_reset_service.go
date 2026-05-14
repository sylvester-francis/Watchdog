package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// ErrInvalidResetToken is returned for any failed reset attempt — expired,
// already-used, or non-existent. Caller MUST NOT differentiate these in the
// user-facing response (prevents enumeration / probing).
var ErrInvalidResetToken = errors.New("invalid or expired reset token")

// Mailer is the minimal interface PasswordResetService needs.
// Implemented by *email.TransactionalSender.
type Mailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// PasswordHasher hashes new passwords. Implemented by *crypto.PasswordHasher.
type PasswordHasher interface {
	Hash(plaintext string) (string, error)
}

// PasswordResetService orchestrates the forgot-password flow: token issuance,
// email delivery, and token consumption + password update.
type PasswordResetService struct {
	tokens ports.PasswordResetTokenRepository
	users  ports.UserRepository
	mailer Mailer
	hasher PasswordHasher
	appURL string
}

func NewPasswordResetService(
	tokens ports.PasswordResetTokenRepository,
	users ports.UserRepository,
	mailer Mailer,
	hasher PasswordHasher,
	appURL string,
) *PasswordResetService {
	return &PasswordResetService{tokens: tokens, users: users, mailer: mailer, hasher: hasher, appURL: appURL}
}

// RequestReset generates a token + emails the reset link.
// Returns nil regardless of whether the email matches a real user — caller MUST
// not differentiate the two cases in any user-facing response (anti-enumeration).
// Real outcomes (no user, mailer failure, DB error) are logged server-side only.
func (s *PasswordResetService) RequestReset(ctx context.Context, email, ipAddress string) error {
	user, err := s.users.GetByEmailGlobal(ctx, email)
	if err != nil {
		slog.Error("password reset: lookup user failed", slog.String("error", err.Error()))
		return nil
	}
	if user == nil {
		slog.Info("password reset: unknown email", slog.String("email", email))
		return nil
	}

	tok, plaintext, err := domain.GeneratePasswordResetToken(user.ID, ipAddress)
	if err != nil {
		slog.Error("password reset: generate token failed", slog.String("error", err.Error()))
		return nil
	}
	if err := s.tokens.Create(ctx, tok); err != nil {
		slog.Error("password reset: store token failed", slog.String("error", err.Error()))
		return nil
	}

	body := fmt.Sprintf(
		"You requested a password reset for your WatchDog account.\n\n"+
			"Open this link to set a new password (expires in 30 minutes):\n"+
			"%s/reset-password?token=%s\n\n"+
			"If you didn't request this, ignore this email — your account is unchanged.\n",
		s.appURL, plaintext,
	)
	if err := s.mailer.Send(ctx, user.Email, "Reset your WatchDog password", body); err != nil {
		slog.Error("password reset: send mail failed",
			slog.String("email", user.Email),
			slog.String("error", err.Error()))
	}
	return nil
}

// CompleteReset validates the token, hashes the new password, marks the token used,
// and updates the user record. Returns ErrInvalidResetToken for any failure path
// that the user-facing API should treat as "the link is no good — request a new one".
func (s *PasswordResetService) CompleteReset(ctx context.Context, plaintext, newPassword string) error {
	hash := domain.HashPasswordResetToken(plaintext)
	tok, err := s.tokens.GetByHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("lookup reset token: %w", err)
	}
	if tok == nil || !tok.IsValid() {
		return ErrInvalidResetToken
	}
	user, err := s.users.GetByID(ctx, tok.UserID)
	if err != nil || user == nil {
		return ErrInvalidResetToken
	}
	hashed, err := s.hasher.Hash(newPassword)
	if err != nil {
		return fmt.Errorf("hash new password: %w", err)
	}
	user.PasswordHash = hashed
	now := time.Now()
	user.PasswordChangedAt = &now // existing session middleware uses this to invalidate old sessions
	if err := s.users.Update(ctx, user); err != nil {
		return fmt.Errorf("update user: %w", err)
	}
	if err := s.tokens.MarkUsed(ctx, tok.ID); err != nil {
		slog.Error("password reset: mark used failed",
			slog.String("token_id", tok.ID.String()),
			slog.String("error", err.Error()))
		// Don't fail the request — password is already updated.
	}
	return nil
}

// ResolveUserByToken returns the user a token belongs to, for callers that need
// to log the user_id alongside an audit event (the token's user_id but verified
// against the user repo).
func (s *PasswordResetService) ResolveUserByToken(ctx context.Context, plaintext string) (*domain.User, error) {
	hash := domain.HashPasswordResetToken(plaintext)
	tok, err := s.tokens.GetByHash(ctx, hash)
	if err != nil || tok == nil {
		return nil, ErrInvalidResetToken
	}
	user, err := s.users.GetByID(ctx, tok.UserID)
	if err != nil || user == nil {
		return nil, ErrInvalidResetToken
	}
	return user, nil
}

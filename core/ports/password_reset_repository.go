package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// PasswordResetTokenRepository is the persistence boundary for PasswordResetToken.
type PasswordResetTokenRepository interface {
	Create(ctx context.Context, token *domain.PasswordResetToken) error
	GetByHash(ctx context.Context, hash string) (*domain.PasswordResetToken, error)
	MarkUsed(ctx context.Context, id uuid.UUID) error
	DeleteExpired(ctx context.Context) (int, error)
}

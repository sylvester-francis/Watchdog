package ports

import (
	"context"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
)

// StatusPageSubscriberRepository persists subscriber rows + token lookups.
type StatusPageSubscriberRepository interface {
	Upsert(ctx context.Context, sub *domain.StatusPageSubscriber) error
	GetByPageAndEmail(ctx context.Context, pageID uuid.UUID, email string) (*domain.StatusPageSubscriber, error)
	GetByTokenHash(ctx context.Context, hash string) (*domain.StatusPageSubscriber, error)
	MarkConfirmed(ctx context.Context, id uuid.UUID) error
	MarkUnsubscribed(ctx context.Context, id uuid.UUID) error
	ListActiveForPage(ctx context.Context, pageID uuid.UUID) ([]*domain.StatusPageSubscriber, error)
}

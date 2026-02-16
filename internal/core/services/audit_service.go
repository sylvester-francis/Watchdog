package services

import (
	"context"
	"log/slog"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/internal/core/domain"
	"github.com/sylvester-francis/watchdog/internal/core/ports"
)

// AuditService logs security-relevant events to the audit log.
type AuditService struct {
	repo   ports.AuditLogRepository
	logger *slog.Logger
}

// NewAuditService creates a new AuditService.
func NewAuditService(repo ports.AuditLogRepository, logger *slog.Logger) *AuditService {
	if logger == nil {
		logger = slog.Default()
	}
	return &AuditService{repo: repo, logger: logger}
}

// LogEvent records an audit event. Errors are logged but never returned
// to avoid disrupting the main operation.
func (s *AuditService) LogEvent(ctx context.Context, userID *uuid.UUID, action domain.AuditAction, ipAddress string, metadata map[string]string) {
	entry := domain.NewAuditLog(userID, action, ipAddress, metadata)

	if err := s.repo.Create(ctx, entry); err != nil {
		s.logger.Error("failed to write audit log",
			slog.String("action", string(action)),
			slog.String("error", err.Error()),
		)
	}
}

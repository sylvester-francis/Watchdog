package services

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
)

// ErrInvalidSubscriberToken is returned when a token doesn't match any row.
// Same error class for missing AND already-consumed — caller MUST not
// differentiate in user-facing responses.
var ErrInvalidSubscriberToken = errors.New("invalid subscriber token")

// StatusPageSubscriberMailer is the minimal mailer interface needed.
// Satisfied by *email.TransactionalSender.
type StatusPageSubscriberMailer interface {
	Send(ctx context.Context, to, subject, body string) error
}

// StatusPageSubscriberService orchestrates the subscribe / confirm / unsubscribe
// / notify flows. Anti-enumerating on Subscribe; idempotent on Confirm + Unsubscribe.
type StatusPageSubscriberService struct {
	repo        ports.StatusPageSubscriberRepository
	statusPages ports.StatusPageRepository // for resolving monitor → pages in OnIncidentOpened
	mailer      StatusPageSubscriberMailer
	appURL      string
}

// NewStatusPageSubscriberService constructs the service. statusPages is
// optional — pass nil if the consumer only needs Subscribe / Confirm /
// Unsubscribe (e.g. tests that don't exercise the incident hook).
func NewStatusPageSubscriberService(
	repo ports.StatusPageSubscriberRepository,
	statusPages ports.StatusPageRepository,
	mailer StatusPageSubscriberMailer,
	appURL string,
) *StatusPageSubscriberService {
	return &StatusPageSubscriberService{repo: repo, statusPages: statusPages, mailer: mailer, appURL: appURL}
}

// Subscribe generates (or refreshes) a subscriber row + sends a confirmation
// email. Returns nil regardless of outcome to prevent enumeration:
//   - email not previously subscribed: row created, confirmation email sent
//   - email subscribed but unconfirmed: token refreshed, new confirmation sent
//   - email already confirmed + active: silent no-op (no second email)
//
// Caller must respond with a generic 200 message in all cases.
func (s *StatusPageSubscriberService) Subscribe(ctx context.Context, pageID uuid.UUID, pageName, email string) error {
	existing, err := s.repo.GetByPageAndEmail(ctx, pageID, email)
	if err != nil {
		slog.Error("subscriber: lookup failed", slog.String("error", err.Error()))
		return nil
	}
	if existing != nil && existing.IsActive() {
		slog.Info("subscriber: already active, no-op", slog.String("page_id", pageID.String()))
		return nil
	}

	sub, plaintext, err := domain.GenerateStatusPageSubscriber(pageID, email)
	if err != nil {
		slog.Error("subscriber: generate token", slog.String("error", err.Error()))
		return nil
	}
	if err := s.repo.Upsert(ctx, sub); err != nil {
		slog.Error("subscriber: upsert", slog.String("error", err.Error()))
		return nil
	}

	confirmURL := fmt.Sprintf("%s/api/v1/public/status-subscriber/confirm?token=%s", s.appURL, plaintext)
	unsubURL := fmt.Sprintf("%s/api/v1/public/status-subscriber/unsubscribe?token=%s", s.appURL, plaintext)
	body := fmt.Sprintf(
		"You requested email notifications for the WatchDog status page \"%s\".\n\n"+
			"Click here to confirm your subscription:\n%s\n\n"+
			"If you didn't request this, ignore this email — you won't receive further messages.\n"+
			"Or unsubscribe immediately: %s\n",
		pageName, confirmURL, unsubURL,
	)
	if err := s.mailer.Send(ctx, email, "Confirm your subscription to "+pageName, body); err != nil {
		slog.Error("subscriber: send confirm mail",
			slog.String("email", email),
			slog.String("error", err.Error()))
	}
	return nil
}

// Confirm activates a pending subscription. Idempotent: a second confirm
// with the same token is a no-op.
func (s *StatusPageSubscriberService) Confirm(ctx context.Context, plaintext string) error {
	hash := domain.HashStatusPageSubscriberToken(plaintext)
	sub, err := s.repo.GetByTokenHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("lookup token: %w", err)
	}
	if sub == nil {
		return ErrInvalidSubscriberToken
	}
	if sub.ConfirmedAt != nil {
		return nil
	}
	return s.repo.MarkConfirmed(ctx, sub.ID)
}

// Unsubscribe deactivates a subscription. Idempotent: a second unsubscribe
// with the same token is a no-op.
func (s *StatusPageSubscriberService) Unsubscribe(ctx context.Context, plaintext string) error {
	hash := domain.HashStatusPageSubscriberToken(plaintext)
	sub, err := s.repo.GetByTokenHash(ctx, hash)
	if err != nil {
		return fmt.Errorf("lookup token: %w", err)
	}
	if sub == nil {
		return ErrInvalidSubscriberToken
	}
	if sub.UnsubscribedAt != nil {
		return nil
	}
	return s.repo.MarkUnsubscribed(ctx, sub.ID)
}

// NotifyIncidentOpened sends one email per active subscriber for the page.
// Each email includes a fresh unsubscribe link — we rotate the token per
// send so the receiver always has a working link without storing plaintext.
// One DB write per subscriber; bounded cost per incident.
//
// Failures are logged per-recipient but don't abort the loop.
func (s *StatusPageSubscriberService) NotifyIncidentOpened(
	ctx context.Context,
	pageID uuid.UUID,
	pageName string,
	monitor *domain.Monitor,
	errorMessage string,
) error {
	subs, err := s.repo.ListActiveForPage(ctx, pageID)
	if err != nil {
		return fmt.Errorf("list active subs: %w", err)
	}
	subject := fmt.Sprintf("[%s] %s is DOWN", pageName, monitor.Name)
	for _, sub := range subs {
		plaintext := s.rotateToken(ctx, sub)
		unsubURL := ""
		if plaintext != "" {
			unsubURL = fmt.Sprintf("%s/api/v1/public/status-subscriber/unsubscribe?token=%s", s.appURL, plaintext)
		}
		body := fmt.Sprintf(
			"%s is currently DOWN on the \"%s\" status page.\n\n"+
				"Target: %s\nError: %s\n\n"+
				"You'll receive a follow-up when it recovers.\n\n"+
				"Unsubscribe: %s\n",
			monitor.Name, pageName, monitor.Target, errorMessage, unsubURL,
		)
		if err := s.mailer.Send(ctx, sub.Email, subject, body); err != nil {
			slog.Error("subscriber: notify failed",
				slog.String("email", sub.Email),
				slog.String("error", err.Error()))
			continue
		}
	}
	return nil
}

// OnIncidentOpened is the hook IncidentService calls when a new incident
// fires. Resolves which status pages contain the monitor, then notifies the
// active subscribers on each page. No-op if statusPages wasn't injected.
// Fire-and-forget from the caller's perspective — failures logged not returned.
func (s *StatusPageSubscriberService) OnIncidentOpened(ctx context.Context, incident *domain.Incident, monitor *domain.Monitor) {
	if s.statusPages == nil || monitor == nil {
		return
	}
	pages, err := s.statusPages.FindPagesByMonitorID(ctx, monitor.ID)
	if err != nil {
		slog.Error("subscriber notify: find pages",
			slog.String("monitor_id", monitor.ID.String()),
			slog.String("error", err.Error()))
		return
	}
	errMsg := ""
	if incident != nil && incident.AlertContext != nil {
		errMsg = incident.AlertContext.ErrorMessage
	}
	for _, page := range pages {
		if err := s.NotifyIncidentOpened(ctx, page.ID, page.Name, monitor, errMsg); err != nil {
			slog.Error("subscriber notify: page fan-out failed",
				slog.String("page_id", page.ID.String()),
				slog.String("error", err.Error()))
		}
	}
}

// rotateToken generates a fresh plaintext token, hashes + persists it on the
// existing row (preserving confirmation + unsubscribe state), returns the
// plaintext for embedding in the outgoing email. Empty string on failure;
// caller embeds an empty unsub link rather than failing the notification.
func (s *StatusPageSubscriberService) rotateToken(ctx context.Context, sub *domain.StatusPageSubscriber) string {
	fresh, plaintext, err := domain.GenerateStatusPageSubscriber(sub.StatusPageID, sub.Email)
	if err != nil {
		return ""
	}
	// Preserve identity + status — only the token rotates.
	fresh.ID = sub.ID
	fresh.ConfirmedAt = sub.ConfirmedAt
	fresh.UnsubscribedAt = sub.UnsubscribedAt
	fresh.CreatedAt = sub.CreatedAt
	if err := s.repo.Upsert(ctx, fresh); err != nil {
		return ""
	}
	return plaintext
}

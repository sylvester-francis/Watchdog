package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/core/domain"
)

type fakeSubRepo struct {
	byID    map[uuid.UUID]*domain.StatusPageSubscriber
	byHash  map[string]*domain.StatusPageSubscriber
	byPgEml map[string]*domain.StatusPageSubscriber // key = pageID|email
}

func newFakeSubRepo() *fakeSubRepo {
	return &fakeSubRepo{
		byID:    map[uuid.UUID]*domain.StatusPageSubscriber{},
		byHash:  map[string]*domain.StatusPageSubscriber{},
		byPgEml: map[string]*domain.StatusPageSubscriber{},
	}
}

func pgEmlKey(pageID uuid.UUID, email string) string { return pageID.String() + "|" + email }

func (f *fakeSubRepo) Upsert(_ context.Context, s *domain.StatusPageSubscriber) error {
	// Mirror the SQL ON CONFLICT (page_id, email) DO UPDATE behavior:
	// if a row already exists for (page, email), keep its ID + status + created_at
	// but refresh token_hash + last_confirmation_sent_at.
	key := pgEmlKey(s.StatusPageID, s.Email)
	if existing, ok := f.byPgEml[key]; ok {
		delete(f.byHash, existing.TokenHash) // old token no longer valid
		existing.TokenHash = s.TokenHash
		existing.LastConfirmationSentAt = s.LastConfirmationSentAt
		f.byHash[existing.TokenHash] = existing
		return nil
	}
	f.byID[s.ID] = s
	f.byHash[s.TokenHash] = s
	f.byPgEml[key] = s
	return nil
}

func (f *fakeSubRepo) GetByPageAndEmail(_ context.Context, pageID uuid.UUID, email string) (*domain.StatusPageSubscriber, error) {
	return f.byPgEml[pgEmlKey(pageID, email)], nil
}

func (f *fakeSubRepo) GetByTokenHash(_ context.Context, hash string) (*domain.StatusPageSubscriber, error) {
	return f.byHash[hash], nil
}

func (f *fakeSubRepo) MarkConfirmed(_ context.Context, id uuid.UUID) error {
	if sub, ok := f.byID[id]; ok {
		now := time.Now()
		sub.ConfirmedAt = &now
	}
	return nil
}

func (f *fakeSubRepo) MarkUnsubscribed(_ context.Context, id uuid.UUID) error {
	if sub, ok := f.byID[id]; ok {
		now := time.Now()
		sub.UnsubscribedAt = &now
	}
	return nil
}

func (f *fakeSubRepo) ListActiveForPage(_ context.Context, pageID uuid.UUID) ([]*domain.StatusPageSubscriber, error) {
	var out []*domain.StatusPageSubscriber
	for _, s := range f.byID {
		if s.StatusPageID == pageID && s.IsActive() {
			out = append(out, s)
		}
	}
	return out, nil
}

type recordingMailer struct {
	calls []recordedSend
	err   error
}

type recordedSend struct{ to, subject, body string }

func (m *recordingMailer) Send(_ context.Context, to, subject, body string) error {
	if m.err != nil {
		return m.err
	}
	m.calls = append(m.calls, recordedSend{to, subject, body})
	return nil
}

func TestSubscribe_FirstTime_SendsConfirmationEmail(t *testing.T) {
	repo := newFakeSubRepo()
	mailer := &recordingMailer{}
	svc := NewStatusPageSubscriberService(repo, nil, mailer, "https://app.test")

	require.NoError(t, svc.Subscribe(context.Background(), uuid.New(), "Page A", "alice@example.com"))
	require.Len(t, mailer.calls, 1)
	assert.Equal(t, "alice@example.com", mailer.calls[0].to)
	assert.Contains(t, mailer.calls[0].subject, "Page A")
	assert.Contains(t, mailer.calls[0].body, "https://app.test/api/v1/public/status-subscriber/confirm?token=wd_sub_")
}

func TestSubscribe_AlreadyActive_IsNoOp(t *testing.T) {
	pageID := uuid.New()
	repo := newFakeSubRepo()
	mailer := &recordingMailer{}
	svc := NewStatusPageSubscriberService(repo, nil, mailer, "https://app.test")

	// First subscribe + confirm.
	require.NoError(t, svc.Subscribe(context.Background(), pageID, "Page A", "bob@example.com"))
	require.Len(t, mailer.calls, 1)
	existing, _ := repo.GetByPageAndEmail(context.Background(), pageID, "bob@example.com")
	require.NoError(t, repo.MarkConfirmed(context.Background(), existing.ID))

	mailer.calls = nil
	require.NoError(t, svc.Subscribe(context.Background(), pageID, "Page A", "bob@example.com"))
	assert.Empty(t, mailer.calls, "second subscribe on active row must not send another email")
}

func TestConfirm_ValidToken(t *testing.T) {
	pageID := uuid.New()
	repo := newFakeSubRepo()
	svc := NewStatusPageSubscriberService(repo, nil, &recordingMailer{}, "https://app.test")

	sub, plaintext, _ := domain.GenerateStatusPageSubscriber(pageID, "carol@example.com")
	repo.Upsert(context.Background(), sub)

	require.NoError(t, svc.Confirm(context.Background(), plaintext))
	got, _ := repo.GetByTokenHash(context.Background(), sub.TokenHash)
	require.NotNil(t, got.ConfirmedAt)
}

func TestConfirm_InvalidToken(t *testing.T) {
	svc := NewStatusPageSubscriberService(newFakeSubRepo(), nil, &recordingMailer{}, "")
	err := svc.Confirm(context.Background(), "wd_sub_garbage")
	assert.True(t, errors.Is(err, ErrInvalidSubscriberToken))
}

func TestConfirm_Idempotent(t *testing.T) {
	pageID := uuid.New()
	repo := newFakeSubRepo()
	svc := NewStatusPageSubscriberService(repo, nil, &recordingMailer{}, "")

	sub, plaintext, _ := domain.GenerateStatusPageSubscriber(pageID, "dan@example.com")
	repo.Upsert(context.Background(), sub)

	require.NoError(t, svc.Confirm(context.Background(), plaintext))
	require.NoError(t, svc.Confirm(context.Background(), plaintext), "second confirm is a no-op")
}

func TestUnsubscribe_Idempotent(t *testing.T) {
	pageID := uuid.New()
	repo := newFakeSubRepo()
	svc := NewStatusPageSubscriberService(repo, nil, &recordingMailer{}, "")

	sub, plaintext, _ := domain.GenerateStatusPageSubscriber(pageID, "eve@example.com")
	repo.Upsert(context.Background(), sub)
	repo.MarkConfirmed(context.Background(), sub.ID)

	require.NoError(t, svc.Unsubscribe(context.Background(), plaintext))
	require.NoError(t, svc.Unsubscribe(context.Background(), plaintext), "second unsub is a no-op")
}

func TestUnsubscribe_InvalidToken(t *testing.T) {
	svc := NewStatusPageSubscriberService(newFakeSubRepo(), nil, &recordingMailer{}, "")
	err := svc.Unsubscribe(context.Background(), "wd_sub_garbage")
	assert.True(t, errors.Is(err, ErrInvalidSubscriberToken))
}

func TestNotifyIncidentOpened_FansOutToActive(t *testing.T) {
	pageID := uuid.New()
	repo := newFakeSubRepo()
	mailer := &recordingMailer{}
	svc := NewStatusPageSubscriberService(repo, nil, mailer, "https://app.test")

	// Three subs: confirmed+active (x2), confirmed+unsubscribed (skipped).
	for _, email := range []string{"active1@x.co", "active2@x.co"} {
		sub, _, _ := domain.GenerateStatusPageSubscriber(pageID, email)
		repo.Upsert(context.Background(), sub)
		repo.MarkConfirmed(context.Background(), sub.ID)
	}
	skipSub, _, _ := domain.GenerateStatusPageSubscriber(pageID, "skip@x.co")
	repo.Upsert(context.Background(), skipSub)
	repo.MarkConfirmed(context.Background(), skipSub.ID)
	repo.MarkUnsubscribed(context.Background(), skipSub.ID)

	mon := &domain.Monitor{Name: "api.example.com", Target: "https://api.example.com/health"}
	err := svc.NotifyIncidentOpened(context.Background(), pageID, "My Page", mon, "HTTP 500")
	require.NoError(t, err)
	assert.Len(t, mailer.calls, 2, "only confirmed-non-unsubbed subscribers get notified")
	for _, call := range mailer.calls {
		assert.Contains(t, call.subject, "My Page")
		assert.Contains(t, call.subject, "api.example.com")
		assert.Contains(t, call.body, "api.example.com")
		assert.Contains(t, call.body, "/unsubscribe?token=wd_sub_")
	}
}

func TestNotifyIncidentOpened_NoSubscribers(t *testing.T) {
	repo := newFakeSubRepo()
	svc := NewStatusPageSubscriberService(repo, nil, &recordingMailer{}, "")

	err := svc.NotifyIncidentOpened(context.Background(), uuid.New(), "Empty Page", &domain.Monitor{Name: "x"}, "")
	assert.NoError(t, err)
}

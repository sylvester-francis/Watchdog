package handlers

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	tracepb "go.opentelemetry.io/proto/otlp/trace/v1"
	"google.golang.org/protobuf/proto"

	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/core/ports"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
)

// stubAPITokenRepo returns a single configured token on GetByTokenHash and
// errors out for everything else. It is enough to drive the APITokenAuth
// middleware in tests without standing up a real database.
type stubAPITokenRepo struct {
	hash  string
	token *domain.APIToken
}

func (s *stubAPITokenRepo) GetByTokenHash(_ context.Context, hash string) (*domain.APIToken, error) {
	if hash != s.hash || s.token == nil {
		return nil, errors.New("not found")
	}
	return s.token, nil
}
func (s *stubAPITokenRepo) Create(context.Context, *domain.APIToken) error { return nil }
func (s *stubAPITokenRepo) GetByUserID(context.Context, uuid.UUID) ([]*domain.APIToken, error) {
	return nil, nil
}
func (s *stubAPITokenRepo) Delete(context.Context, uuid.UUID) error              { return nil }
func (s *stubAPITokenRepo) UpdateLastUsed(context.Context, uuid.UUID, string) error { return nil }

// stubResolver implements ports.TenantResolver with a fixed mapping from
// userID to tenant_id. Resolve() is the fallback path; we only set
// TenantID() because that's what TenantScope calls for authenticated users.
type stubResolver struct {
	tenantByUser map[uuid.UUID]string
}

func (s *stubResolver) Resolve(context.Context) string { return "default" }
func (s *stubResolver) TenantID(_ context.Context, userID uuid.UUID) (string, error) {
	if t, ok := s.tenantByUser[userID]; ok {
		return t, nil
	}
	return "default", nil
}

// TestTracesHandler_FullMiddlewareChain_StampsUserAndTenant proves that the
// production middleware chain wired up at /v1/traces by router.go
// (APITokenAuth → RequireScope → TenantScope → TracesHandler) actually
// delivers user_id and tenant_id to the handler, which then stamps them on
// the persisted spans. This is the integration boundary that catches the
// "router forgot to mount tenantMW on the OTLP group" class of bug.
func TestTracesHandler_FullMiddlewareChain_StampsUserAndTenant(t *testing.T) {
	plaintext := "wd_test_token_abc123"
	userID := uuid.MustParse("dddddddd-dddd-dddd-dddd-dddddddddddd")
	const tenantID = "acme-corp"

	tokenRepo := &stubAPITokenRepo{
		hash: domain.HashToken(plaintext),
		token: &domain.APIToken{
			ID:     uuid.New(),
			UserID: userID,
			Scope:  domain.TokenScopeTelemetryIngest,
		},
	}
	resolver := &stubResolver{tenantByUser: map[uuid.UUID]string{userID: tenantID}}
	repo := &fakeSpanRepo{}

	e := echo.New()
	otlp := e.Group("/v1")
	otlp.Use(middleware.APITokenAuth(tokenRepo))
	otlp.Use(middleware.RequireScope(domain.TokenScopeTelemetryIngest))
	otlp.Use(middleware.TenantScope(resolver))
	h := NewTracesHandler(repo, slog.New(slog.NewTextHandler(&bytes.Buffer{}, nil)))
	otlp.POST("/traces", h.Handle)

	body, err := proto.Marshal(requestWithSpan(&tracepb.Span{
		TraceId: bytes16(0xAB), SpanId: bytes8(0xCD),
		Name: "GET /healthz", StartTimeUnixNano: 1, EndTimeUnixNano: 2,
	}, "svc"))
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v1/traces", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/x-protobuf")
	req.Header.Set("Authorization", "Bearer "+plaintext)
	rec := httptest.NewRecorder()
	e.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code, "auth + scope + tenant chain should accept the request")
	require.Len(t, repo.inserted, 1)
	assert.Equal(t, userID, repo.inserted[0].UserID, "user_id is stamped from the token")
	assert.Equal(t, tenantID, repo.inserted[0].TenantID, "tenant_id is stamped from the resolver")
}

// Compile-time assertions that our stubs satisfy the ports they replace.
var (
	_ ports.APITokenRepository = (*stubAPITokenRepo)(nil)
	_ ports.TenantResolver     = (*stubResolver)(nil)
)

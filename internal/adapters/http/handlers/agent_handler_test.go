package handlers_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/gorilla/sessions"
	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/sylvester-francis/watchdog/internal/adapters/http/handlers"
	"github.com/sylvester-francis/watchdog/internal/adapters/http/middleware"
	"github.com/sylvester-francis/watchdog/core/domain"
	"github.com/sylvester-francis/watchdog/internal/testutil/mocks"
)

func newAgentTestContext(method, path string, form url.Values, userID uuid.UUID) (echo.Context, *httptest.ResponseRecorder) {
	var body *strings.Reader
	if form != nil {
		body = strings.NewReader(form.Encode())
	} else {
		body = strings.NewReader("")
	}
	req := httptest.NewRequest(method, path, body)
	if form != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}
	rec := httptest.NewRecorder()

	e := echo.New()
	e.Renderer = &noOpRenderer{}
	c := e.NewContext(req, rec)

	store := sessions.NewCookieStore([]byte("test-session-secret-at-least-32-bytes!"))
	c.Set("session_store", store)

	if userID != uuid.Nil {
		c.Set(middleware.UserIDKey, userID.String())
	}

	return c, rec
}

func TestAgentCreate_EmptyName_Returns400(t *testing.T) {
	userID := uuid.New()
	agentAuthSvc := &mocks.MockAgentAuthService{}
	h := handlers.NewAgentHandler(agentAuthSvc, &mocks.MockAgentRepository{}, nil, nil)

	c, rec := newAgentTestContext(http.MethodPost, "/agents", url.Values{
		"name": {""},
	}, userID)

	err := h.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAgentCreate_NameTooLong_Returns400(t *testing.T) {
	userID := uuid.New()
	agentAuthSvc := &mocks.MockAgentAuthService{}
	h := handlers.NewAgentHandler(agentAuthSvc, &mocks.MockAgentRepository{}, nil, nil)

	longName := strings.Repeat("a", 256)
	c, rec := newAgentTestContext(http.MethodPost, "/agents", url.Values{
		"name": {longName},
	}, userID)

	err := h.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAgentCreate_NameExactly255_Succeeds(t *testing.T) {
	// Boundary test: exactly 255 chars should be accepted (kills > vs >= mutant)
	userID := uuid.New()
	agentAuthSvc := &mocks.MockAgentAuthService{
		CreateAgentFn: func(_ context.Context, _ string, name string) (*domain.Agent, string, error) {
			return &domain.Agent{
				ID:     uuid.New(),
				UserID: userID,
				Name:   name,
			}, uuid.New().String() + ":secret", nil
		},
	}
	h := handlers.NewAgentHandler(agentAuthSvc, &mocks.MockAgentRepository{}, nil, nil)

	name255 := strings.Repeat("a", 255)
	c, rec := newAgentTestContext(http.MethodPost, "/agents", url.Values{
		"name": {name255},
	}, userID)

	err := h.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
}

func TestAgentCreate_Success_JSON(t *testing.T) {
	userID := uuid.New()
	agentAuthSvc := &mocks.MockAgentAuthService{
		CreateAgentFn: func(_ context.Context, uid string, name string) (*domain.Agent, string, error) {
			return &domain.Agent{
				ID:     uuid.New(),
				UserID: userID,
				Name:   name,
			}, uuid.New().String() + ":secret", nil
		},
	}
	h := handlers.NewAgentHandler(agentAuthSvc, &mocks.MockAgentRepository{}, nil, nil)

	c, rec := newAgentTestContext(http.MethodPost, "/agents", url.Values{
		"name": {"my-agent"},
	}, userID)

	err := h.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusCreated, rec.Code)
	assert.Contains(t, rec.Body.String(), "my-agent")
	assert.Contains(t, rec.Body.String(), "api_key")
}

func TestAgentCreate_XSS_NameEscaped(t *testing.T) {
	userID := uuid.New()
	agentAuthSvc := &mocks.MockAgentAuthService{
		CreateAgentFn: func(_ context.Context, _ string, name string) (*domain.Agent, string, error) {
			return &domain.Agent{
				ID:     uuid.New(),
				UserID: userID,
				Name:   name,
			}, uuid.New().String() + ":secret", nil
		},
	}
	h := handlers.NewAgentHandler(agentAuthSvc, &mocks.MockAgentRepository{}, nil, nil)

	c, rec := newAgentTestContext(http.MethodPost, "/agents", url.Values{
		"name": {`<script>alert("xss")</script>`},
	}, userID)
	c.Request().Header.Set("HX-Request", "true")

	err := h.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, rec.Code)

	body := rec.Body.String()
	assert.NotContains(t, body, "<script>")
	assert.Contains(t, body, "&lt;script&gt;")
}

func TestAgentDelete_InvalidID_Returns400(t *testing.T) {
	userID := uuid.New()
	h := handlers.NewAgentHandler(&mocks.MockAgentAuthService{}, &mocks.MockAgentRepository{}, nil, nil)

	c, rec := newAgentTestContext(http.MethodDelete, "/agents/not-a-uuid", nil, userID)
	c.SetParamNames("id")
	c.SetParamValues("not-a-uuid")

	err := h.Delete(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestAgentDelete_NotOwned_Returns404(t *testing.T) {
	userID := uuid.New()
	otherUserID := uuid.New()
	agentID := uuid.New()

	agentRepo := &mocks.MockAgentRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Agent, error) {
			return &domain.Agent{
				ID:     agentID,
				UserID: otherUserID, // Different user
			}, nil
		},
	}
	h := handlers.NewAgentHandler(&mocks.MockAgentAuthService{}, agentRepo, nil, nil)

	c, rec := newAgentTestContext(http.MethodDelete, "/agents/"+agentID.String(), nil, userID)
	c.SetParamNames("id")
	c.SetParamValues(agentID.String())

	err := h.Delete(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, rec.Code)
}

func TestAgentDelete_Success(t *testing.T) {
	userID := uuid.New()
	agentID := uuid.New()
	deleted := false

	agentRepo := &mocks.MockAgentRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Agent, error) {
			return &domain.Agent{
				ID:     agentID,
				UserID: userID,
			}, nil
		},
		DeleteFn: func(_ context.Context, id uuid.UUID) error {
			assert.Equal(t, agentID, id)
			deleted = true
			return nil
		},
	}
	h := handlers.NewAgentHandler(&mocks.MockAgentAuthService{}, agentRepo, nil, nil)

	c, rec := newAgentTestContext(http.MethodDelete, "/agents/"+agentID.String(), nil, userID)
	c.SetParamNames("id")
	c.SetParamValues(agentID.String())

	err := h.Delete(c)
	require.NoError(t, err)
	assert.True(t, deleted)
	// Should redirect to dashboard
	assert.Equal(t, http.StatusFound, rec.Code)
}

func TestAgentCreate_Unauthorized(t *testing.T) {
	h := handlers.NewAgentHandler(&mocks.MockAgentAuthService{}, &mocks.MockAgentRepository{}, nil, nil)

	c, rec := newAgentTestContext(http.MethodPost, "/agents", url.Values{
		"name": {"my-agent"},
	}, uuid.Nil) // No user ID set

	err := h.Create(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

package handlers_test

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

func newIncidentTestContext(method, path string, userID uuid.UUID) (echo.Context, *httptest.ResponseRecorder) {
	req := httptest.NewRequest(method, path, nil)
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

func TestIncidentAcknowledge_Success(t *testing.T) {
	userID := uuid.New()
	incidentID := uuid.New()
	monitorID := uuid.New()
	agentID := uuid.New()
	acked := false

	incidentSvc := &mocks.MockIncidentService{
		GetIncidentFn: func(_ context.Context, id uuid.UUID) (*domain.Incident, error) {
			return &domain.Incident{ID: id, MonitorID: monitorID}, nil
		},
		AcknowledgeIncidentFn: func(_ context.Context, id uuid.UUID, uid uuid.UUID) error {
			assert.Equal(t, incidentID, id)
			assert.Equal(t, userID, uid)
			acked = true
			return nil
		},
	}

	h := handlers.NewIncidentHandler(incidentSvc, &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return &domain.Monitor{ID: monitorID, AgentID: agentID}, nil
		},
	}, &mocks.MockAgentRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Agent, error) {
			return &domain.Agent{ID: agentID, UserID: userID}, nil
		},
	}, nil, nil)

	c, rec := newIncidentTestContext(http.MethodPost, "/incidents/"+incidentID.String()+"/ack", userID)
	c.SetParamNames("id")
	c.SetParamValues(incidentID.String())

	err := h.Acknowledge(c)
	require.NoError(t, err)
	assert.True(t, acked)
	// Non-HTMX request -> redirect
	assert.Equal(t, http.StatusFound, rec.Code)
}

func TestIncidentAcknowledge_Unauthorized(t *testing.T) {
	h := handlers.NewIncidentHandler(&mocks.MockIncidentService{}, &mocks.MockMonitorRepository{}, &mocks.MockAgentRepository{}, nil, nil)

	c, rec := newIncidentTestContext(http.MethodPost, "/incidents/"+uuid.New().String()+"/ack", uuid.Nil)
	c.SetParamNames("id")
	c.SetParamValues(uuid.New().String())

	err := h.Acknowledge(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusUnauthorized, rec.Code)
}

func TestIncidentAcknowledge_InvalidID(t *testing.T) {
	userID := uuid.New()
	h := handlers.NewIncidentHandler(&mocks.MockIncidentService{}, &mocks.MockMonitorRepository{}, &mocks.MockAgentRepository{}, nil, nil)

	c, rec := newIncidentTestContext(http.MethodPost, "/incidents/bad-id/ack", userID)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	err := h.Acknowledge(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestIncidentResolve_Success(t *testing.T) {
	userID := uuid.New()
	incidentID := uuid.New()
	monitorID := uuid.New()
	agentID := uuid.New()
	resolved := false

	incidentSvc := &mocks.MockIncidentService{
		GetIncidentFn: func(_ context.Context, id uuid.UUID) (*domain.Incident, error) {
			return &domain.Incident{ID: id, MonitorID: monitorID}, nil
		},
		ResolveIncidentFn: func(_ context.Context, id uuid.UUID) error {
			assert.Equal(t, incidentID, id)
			resolved = true
			return nil
		},
	}

	h := handlers.NewIncidentHandler(incidentSvc, &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return &domain.Monitor{ID: monitorID, AgentID: agentID}, nil
		},
	}, &mocks.MockAgentRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Agent, error) {
			return &domain.Agent{ID: agentID, UserID: userID}, nil
		},
	}, nil, nil)

	c, rec := newIncidentTestContext(http.MethodPost, "/incidents/"+incidentID.String()+"/resolve", userID)
	c.SetParamNames("id")
	c.SetParamValues(incidentID.String())

	err := h.Resolve(c)
	require.NoError(t, err)
	assert.True(t, resolved)
	assert.Equal(t, http.StatusFound, rec.Code)
}

func TestIncidentResolve_ServiceError(t *testing.T) {
	userID := uuid.New()
	incidentID := uuid.New()
	monitorID := uuid.New()
	agentID := uuid.New()

	incidentSvc := &mocks.MockIncidentService{
		GetIncidentFn: func(_ context.Context, id uuid.UUID) (*domain.Incident, error) {
			return &domain.Incident{ID: id, MonitorID: monitorID}, nil
		},
		ResolveIncidentFn: func(_ context.Context, _ uuid.UUID) error {
			return errors.New("db error")
		},
	}

	h := handlers.NewIncidentHandler(incidentSvc, &mocks.MockMonitorRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Monitor, error) {
			return &domain.Monitor{ID: monitorID, AgentID: agentID}, nil
		},
	}, &mocks.MockAgentRepository{
		GetByIDFn: func(_ context.Context, _ uuid.UUID) (*domain.Agent, error) {
			return &domain.Agent{ID: agentID, UserID: userID}, nil
		},
	}, nil, nil)

	c, rec := newIncidentTestContext(http.MethodPost, "/incidents/"+incidentID.String()+"/resolve", userID)
	c.SetParamNames("id")
	c.SetParamValues(incidentID.String())

	err := h.Resolve(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusInternalServerError, rec.Code)
}

func TestIncidentResolve_InvalidID(t *testing.T) {
	userID := uuid.New()
	h := handlers.NewIncidentHandler(&mocks.MockIncidentService{}, &mocks.MockMonitorRepository{}, &mocks.MockAgentRepository{}, nil, nil)

	c, rec := newIncidentTestContext(http.MethodPost, "/incidents/bad-id/resolve", userID)
	c.SetParamNames("id")
	c.SetParamValues("bad-id")

	err := h.Resolve(c)
	require.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, rec.Code)
}

package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAuditLog(t *testing.T) {
	userID := uuid.New()
	action := AuditLoginSuccess
	ipAddress := "192.168.1.1"
	metadata := map[string]string{"browser": "Chrome", "os": "Linux"}

	log := NewAuditLog(&userID, action, ipAddress, metadata)

	require.NotNil(t, log)
	assert.NotEqual(t, uuid.Nil, log.ID, "ID should be a generated UUID")
	require.NotNil(t, log.UserID)
	assert.Equal(t, userID, *log.UserID)
	assert.Equal(t, action, log.Action)
	assert.Equal(t, ipAddress, log.IPAddress)
	assert.Equal(t, metadata, log.Metadata)
	assert.False(t, log.CreatedAt.IsZero(), "CreatedAt should be set")
}

func TestNewAuditLog_NilUserID(t *testing.T) {
	action := AuditLoginFailed
	ipAddress := "10.0.0.1"
	metadata := map[string]string{"reason": "invalid_password"}

	log := NewAuditLog(nil, action, ipAddress, metadata)

	require.NotNil(t, log)
	assert.Nil(t, log.UserID, "UserID should be nil for unauthenticated events")
	assert.Equal(t, action, log.Action)
	assert.Equal(t, ipAddress, log.IPAddress)
}

func TestNewAuditLog_NilMetadata(t *testing.T) {
	userID := uuid.New()

	log := NewAuditLog(&userID, AuditSettingsChanged, "127.0.0.1", nil)

	require.NotNil(t, log)
	assert.Nil(t, log.Metadata, "Metadata should be nil when not provided")
}

func TestAuditAction_Constants(t *testing.T) {
	tests := []struct {
		name   string
		action AuditAction
		want   string
	}{
		{"login success", AuditLoginSuccess, "login_success"},
		{"login failed", AuditLoginFailed, "login_failed"},
		{"API token created", AuditAPITokenCreated, "api_token_created"},
		{"API token revoked", AuditAPITokenRevoked, "api_token_revoked"},
		{"agent created", AuditAgentCreated, "agent_created"},
		{"agent deleted", AuditAgentDeleted, "agent_deleted"},
		{"monitor created", AuditMonitorCreated, "monitor_created"},
		{"monitor updated", AuditMonitorUpdated, "monitor_updated"},
		{"monitor deleted", AuditMonitorDeleted, "monitor_deleted"},
		{"incident acknowledged", AuditIncidentAcked, "incident_acknowledged"},
		{"incident resolved", AuditIncidentResolved, "incident_resolved"},
		{"settings changed", AuditSettingsChanged, "settings_changed"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, string(tt.action))
		})
	}
}

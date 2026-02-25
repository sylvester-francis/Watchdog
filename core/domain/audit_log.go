package domain

import (
	"time"

	"github.com/google/uuid"
)

// AuditAction represents the type of audited event.
type AuditAction string

const (
	AuditLoginSuccess      AuditAction = "login_success"
	AuditLoginFailed       AuditAction = "login_failed"
	AuditAPITokenCreated   AuditAction = "api_token_created"
	AuditAPITokenRevoked   AuditAction = "api_token_revoked"
	AuditAgentCreated      AuditAction = "agent_created"
	AuditAgentDeleted      AuditAction = "agent_deleted"
	AuditMonitorCreated    AuditAction = "monitor_created"
	AuditMonitorUpdated    AuditAction = "monitor_updated"
	AuditMonitorDeleted    AuditAction = "monitor_deleted"
	AuditIncidentAcked     AuditAction = "incident_acknowledged"
	AuditIncidentResolved  AuditAction = "incident_resolved"
	AuditSettingsChanged       AuditAction = "settings_changed"
	AuditPasswordResetByAdmin  AuditAction = "password_reset_by_admin"
	AuditPasswordChanged       AuditAction = "password_changed"
	AuditRegisterSuccess       AuditAction = "register_success"
	AuditRegisterBlocked       AuditAction = "register_blocked"
	AuditUserDeleted           AuditAction = "user_deleted"
	AuditLogout                AuditAction = "logout"
	AuditChannelCreated        AuditAction = "channel_created"
	AuditChannelDeleted        AuditAction = "channel_deleted"
)

// AuditLog represents a security audit event.
type AuditLog struct {
	ID        uuid.UUID
	UserID    *uuid.UUID
	Action    AuditAction
	Metadata  map[string]string
	IPAddress string
	CreatedAt time.Time
}

// NewAuditLog creates a new audit log entry.
func NewAuditLog(userID *uuid.UUID, action AuditAction, ipAddress string, metadata map[string]string) *AuditLog {
	return &AuditLog{
		ID:        uuid.New(),
		UserID:    userID,
		Action:    action,
		Metadata:  metadata,
		IPAddress: ipAddress,
		CreatedAt: time.Now(),
	}
}

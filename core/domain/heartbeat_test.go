package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHeartbeatStatus_IsValid(t *testing.T) {
	tests := []struct {
		status HeartbeatStatus
		want   bool
	}{
		{HeartbeatStatusUp, true},
		{HeartbeatStatusDown, true},
		{HeartbeatStatusTimeout, true},
		{HeartbeatStatusError, true},
		{HeartbeatStatus("invalid"), false},
		{HeartbeatStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHeartbeatStatus_IsSuccess(t *testing.T) {
	tests := []struct {
		status HeartbeatStatus
		want   bool
	}{
		{HeartbeatStatusUp, true},
		{HeartbeatStatusDown, false},
		{HeartbeatStatusTimeout, false},
		{HeartbeatStatusError, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsSuccess()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHeartbeatStatus_IsFailure(t *testing.T) {
	tests := []struct {
		status HeartbeatStatus
		want   bool
	}{
		{HeartbeatStatusUp, false},
		{HeartbeatStatusDown, true},
		{HeartbeatStatusTimeout, true},
		{HeartbeatStatusError, true},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsFailure()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewHeartbeat(t *testing.T) {
	monitorID := uuid.New()
	agentID := uuid.New()
	status := HeartbeatStatusUp

	heartbeat := NewHeartbeat(monitorID, agentID, status)

	require.NotNil(t, heartbeat)
	assert.False(t, heartbeat.Time.IsZero())
	assert.Equal(t, monitorID, heartbeat.MonitorID)
	assert.Equal(t, agentID, heartbeat.AgentID)
	assert.Equal(t, status, heartbeat.Status)
	assert.Nil(t, heartbeat.LatencyMs)
	assert.Nil(t, heartbeat.ErrorMessage)
}

func TestNewSuccessHeartbeat(t *testing.T) {
	monitorID := uuid.New()
	agentID := uuid.New()
	latencyMs := 42

	heartbeat := NewSuccessHeartbeat(monitorID, agentID, latencyMs)

	require.NotNil(t, heartbeat)
	assert.Equal(t, HeartbeatStatusUp, heartbeat.Status)
	require.NotNil(t, heartbeat.LatencyMs)
	assert.Equal(t, latencyMs, *heartbeat.LatencyMs)
	assert.Nil(t, heartbeat.ErrorMessage)
}

func TestNewFailureHeartbeat(t *testing.T) {
	monitorID := uuid.New()
	agentID := uuid.New()
	status := HeartbeatStatusDown
	errorMsg := "connection refused"

	heartbeat := NewFailureHeartbeat(monitorID, agentID, status, errorMsg)

	require.NotNil(t, heartbeat)
	assert.Equal(t, status, heartbeat.Status)
	require.NotNil(t, heartbeat.ErrorMessage)
	assert.Equal(t, errorMsg, *heartbeat.ErrorMessage)
}

func TestHeartbeat_SetLatency(t *testing.T) {
	heartbeat := NewHeartbeat(uuid.New(), uuid.New(), HeartbeatStatusUp)
	assert.Nil(t, heartbeat.LatencyMs)

	heartbeat.SetLatency(100)

	require.NotNil(t, heartbeat.LatencyMs)
	assert.Equal(t, 100, *heartbeat.LatencyMs)
}

func TestHeartbeat_SetError(t *testing.T) {
	heartbeat := NewHeartbeat(uuid.New(), uuid.New(), HeartbeatStatusDown)
	assert.Nil(t, heartbeat.ErrorMessage)

	heartbeat.SetError("connection timeout")

	require.NotNil(t, heartbeat.ErrorMessage)
	assert.Equal(t, "connection timeout", *heartbeat.ErrorMessage)
}

func TestHeartbeat_HasLatency(t *testing.T) {
	heartbeat := NewHeartbeat(uuid.New(), uuid.New(), HeartbeatStatusUp)
	assert.False(t, heartbeat.HasLatency())

	heartbeat.SetLatency(50)
	assert.True(t, heartbeat.HasLatency())
}

func TestHeartbeat_HasError(t *testing.T) {
	tests := []struct {
		name     string
		errorMsg *string
		want     bool
	}{
		{"nil error", nil, false},
		{"empty error", strPtr(""), false},
		{"valid error", strPtr("connection refused"), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			heartbeat := NewHeartbeat(uuid.New(), uuid.New(), HeartbeatStatusDown)
			heartbeat.ErrorMessage = tt.errorMsg

			got := heartbeat.HasError()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHeartbeat_IsSuccessAndIsFailure(t *testing.T) {
	successHeartbeat := NewHeartbeat(uuid.New(), uuid.New(), HeartbeatStatusUp)
	assert.True(t, successHeartbeat.IsSuccess())
	assert.False(t, successHeartbeat.IsFailure())

	failureHeartbeat := NewHeartbeat(uuid.New(), uuid.New(), HeartbeatStatusDown)
	assert.False(t, failureHeartbeat.IsSuccess())
	assert.True(t, failureHeartbeat.IsFailure())
}

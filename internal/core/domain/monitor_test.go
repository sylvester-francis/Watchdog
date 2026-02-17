package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMonitorType_IsValid(t *testing.T) {
	tests := []struct {
		typ  MonitorType
		want bool
	}{
		{MonitorTypePing, true},
		{MonitorTypeHTTP, true},
		{MonitorTypeTCP, true},
		{MonitorTypeDNS, true},
		{MonitorType("invalid"), false},
		{MonitorType(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.typ), func(t *testing.T) {
			got := tt.typ.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMonitorStatus_IsValid(t *testing.T) {
	tests := []struct {
		status MonitorStatus
		want   bool
	}{
		{MonitorStatusPending, true},
		{MonitorStatusUp, true},
		{MonitorStatusDown, true},
		{MonitorStatusDegraded, true},
		{MonitorStatus("invalid"), false},
		{MonitorStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestMonitorStatus_IsHealthy(t *testing.T) {
	tests := []struct {
		status MonitorStatus
		want   bool
	}{
		{MonitorStatusPending, false},
		{MonitorStatusUp, true},
		{MonitorStatusDown, false},
		{MonitorStatusDegraded, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsHealthy()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewMonitor(t *testing.T) {
	agentID := uuid.New()
	name := "test-monitor"
	monitorType := MonitorTypeHTTP
	target := "https://example.com"

	monitor := NewMonitor(agentID, name, monitorType, target)

	require.NotNil(t, monitor)
	assert.NotEqual(t, monitor.ID.String(), "00000000-0000-0000-0000-000000000000")
	assert.Equal(t, agentID, monitor.AgentID)
	assert.Equal(t, name, monitor.Name)
	assert.Equal(t, monitorType, monitor.Type)
	assert.Equal(t, target, monitor.Target)
	assert.Equal(t, DefaultIntervalSeconds, monitor.IntervalSeconds)
	assert.Equal(t, DefaultTimeoutSeconds, monitor.TimeoutSeconds)
	assert.Equal(t, MonitorStatusPending, monitor.Status)
	assert.True(t, monitor.Enabled)
	assert.False(t, monitor.CreatedAt.IsZero())
}

func TestMonitor_SetInterval(t *testing.T) {
	tests := []struct {
		name     string
		interval int
		want     bool
	}{
		{"below minimum", MinIntervalSeconds - 1, false},
		{"at minimum", MinIntervalSeconds, true},
		{"normal value", 60, true},
		{"at maximum", MaxIntervalSeconds, true},
		{"above maximum", MaxIntervalSeconds + 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := NewMonitor(uuid.New(), "test", MonitorTypeHTTP, "https://example.com")
			originalInterval := monitor.IntervalSeconds

			got := monitor.SetInterval(tt.interval)

			assert.Equal(t, tt.want, got)
			if tt.want {
				assert.Equal(t, tt.interval, monitor.IntervalSeconds)
			} else {
				assert.Equal(t, originalInterval, monitor.IntervalSeconds)
			}
		})
	}
}

func TestMonitor_SetTimeout(t *testing.T) {
	tests := []struct {
		name    string
		timeout int
		want    bool
	}{
		{"below minimum", MinTimeoutSeconds - 1, false},
		{"at minimum", MinTimeoutSeconds, true},
		{"normal value", 30, true},
		{"at maximum", MaxTimeoutSeconds, true},
		{"above maximum", MaxTimeoutSeconds + 1, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			monitor := NewMonitor(uuid.New(), "test", MonitorTypeHTTP, "https://example.com")
			originalTimeout := monitor.TimeoutSeconds

			got := monitor.SetTimeout(tt.timeout)

			assert.Equal(t, tt.want, got)
			if tt.want {
				assert.Equal(t, tt.timeout, monitor.TimeoutSeconds)
			} else {
				assert.Equal(t, originalTimeout, monitor.TimeoutSeconds)
			}
		})
	}
}

func TestMonitor_EnableDisable(t *testing.T) {
	monitor := NewMonitor(uuid.New(), "test", MonitorTypeHTTP, "https://example.com")
	assert.True(t, monitor.IsEnabled())

	monitor.Disable()
	assert.False(t, monitor.IsEnabled())
	assert.False(t, monitor.Enabled)

	monitor.Enable()
	assert.True(t, monitor.IsEnabled())
	assert.True(t, monitor.Enabled)
}

func TestMonitor_UpdateStatus(t *testing.T) {
	monitor := NewMonitor(uuid.New(), "test", MonitorTypeHTTP, "https://example.com")
	assert.Equal(t, MonitorStatusPending, monitor.Status)

	monitor.UpdateStatus(MonitorStatusUp)
	assert.Equal(t, MonitorStatusUp, monitor.Status)

	monitor.UpdateStatus(MonitorStatusDown)
	assert.Equal(t, MonitorStatusDown, monitor.Status)
}

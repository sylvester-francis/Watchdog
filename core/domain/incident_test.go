package domain

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestIncidentStatus_IsValid(t *testing.T) {
	tests := []struct {
		status IncidentStatus
		want   bool
	}{
		{IncidentStatusOpen, true},
		{IncidentStatusAcknowledged, true},
		{IncidentStatusResolved, true},
		{IncidentStatus("invalid"), false},
		{IncidentStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIncidentStatus_IsActive(t *testing.T) {
	tests := []struct {
		status IncidentStatus
		want   bool
	}{
		{IncidentStatusOpen, true},
		{IncidentStatusAcknowledged, true},
		{IncidentStatusResolved, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsActive()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewIncident(t *testing.T) {
	monitorID := uuid.New()

	incident := NewIncident(monitorID)

	require.NotNil(t, incident)
	assert.NotEqual(t, incident.ID.String(), "00000000-0000-0000-0000-000000000000")
	assert.Equal(t, monitorID, incident.MonitorID)
	assert.False(t, incident.StartedAt.IsZero())
	assert.Nil(t, incident.ResolvedAt)
	assert.Nil(t, incident.TTRSeconds)
	assert.Nil(t, incident.AcknowledgedBy)
	assert.Nil(t, incident.AcknowledgedAt)
	assert.Equal(t, IncidentStatusOpen, incident.Status)
	assert.False(t, incident.CreatedAt.IsZero())
}

func TestIncident_Acknowledge(t *testing.T) {
	t.Run("acknowledge open incident", func(t *testing.T) {
		incident := NewIncident(uuid.New())
		userID := uuid.New()

		err := incident.Acknowledge(userID)

		require.NoError(t, err)
		assert.Equal(t, IncidentStatusAcknowledged, incident.Status)
		require.NotNil(t, incident.AcknowledgedBy)
		assert.Equal(t, userID, *incident.AcknowledgedBy)
		require.NotNil(t, incident.AcknowledgedAt)
		assert.False(t, incident.AcknowledgedAt.IsZero())
	})

	t.Run("acknowledge already acknowledged incident", func(t *testing.T) {
		incident := NewIncident(uuid.New())
		userID := uuid.New()
		_ = incident.Acknowledge(userID)

		err := incident.Acknowledge(uuid.New())

		assert.ErrorIs(t, err, ErrIncidentAlreadyAcknowledged)
	})

	t.Run("acknowledge resolved incident", func(t *testing.T) {
		incident := NewIncident(uuid.New())
		_ = incident.Resolve()

		err := incident.Acknowledge(uuid.New())

		assert.ErrorIs(t, err, ErrIncidentAlreadyResolved)
	})
}

func TestIncident_Resolve(t *testing.T) {
	t.Run("resolve open incident", func(t *testing.T) {
		incident := NewIncident(uuid.New())
		time.Sleep(10 * time.Millisecond) // Ensure some time passes

		err := incident.Resolve()

		require.NoError(t, err)
		assert.Equal(t, IncidentStatusResolved, incident.Status)
		require.NotNil(t, incident.ResolvedAt)
		assert.False(t, incident.ResolvedAt.IsZero())
		require.NotNil(t, incident.TTRSeconds)
		assert.GreaterOrEqual(t, *incident.TTRSeconds, 0)
	})

	t.Run("resolve acknowledged incident", func(t *testing.T) {
		incident := NewIncident(uuid.New())
		_ = incident.Acknowledge(uuid.New())

		err := incident.Resolve()

		require.NoError(t, err)
		assert.Equal(t, IncidentStatusResolved, incident.Status)
	})

	t.Run("resolve already resolved incident", func(t *testing.T) {
		incident := NewIncident(uuid.New())
		_ = incident.Resolve()

		err := incident.Resolve()

		assert.ErrorIs(t, err, ErrIncidentAlreadyResolved)
	})
}

func TestIncident_Duration(t *testing.T) {
	t.Run("active incident", func(t *testing.T) {
		incident := NewIncident(uuid.New())
		time.Sleep(10 * time.Millisecond)

		duration := incident.Duration()

		assert.GreaterOrEqual(t, duration.Milliseconds(), int64(10))
	})

	t.Run("resolved incident", func(t *testing.T) {
		incident := NewIncident(uuid.New())
		time.Sleep(10 * time.Millisecond)
		_ = incident.Resolve()

		duration := incident.Duration()

		assert.GreaterOrEqual(t, duration.Milliseconds(), int64(10))
	})
}

func TestIncident_StatusChecks(t *testing.T) {
	incident := NewIncident(uuid.New())

	// Initially open
	assert.True(t, incident.IsOpen())
	assert.False(t, incident.IsAcknowledged())
	assert.False(t, incident.IsResolved())
	assert.True(t, incident.IsActive())

	// After acknowledge
	_ = incident.Acknowledge(uuid.New())
	assert.False(t, incident.IsOpen())
	assert.True(t, incident.IsAcknowledged())
	assert.False(t, incident.IsResolved())
	assert.True(t, incident.IsActive())

	// After resolve
	_ = incident.Resolve()
	assert.False(t, incident.IsOpen())
	assert.False(t, incident.IsAcknowledged()) // Status is now resolved, not acknowledged
	assert.True(t, incident.IsResolved())
	assert.False(t, incident.IsActive())
}

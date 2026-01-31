package domain

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAgentStatus_IsValid(t *testing.T) {
	tests := []struct {
		status AgentStatus
		want   bool
	}{
		{AgentStatusOnline, true},
		{AgentStatusOffline, true},
		{AgentStatus("invalid"), false},
		{AgentStatus(""), false},
	}

	for _, tt := range tests {
		t.Run(string(tt.status), func(t *testing.T) {
			got := tt.status.IsValid()
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestNewAgent(t *testing.T) {
	userID := uuid.New()
	name := "test-agent"
	apiKey := []byte("encrypted_key")

	agent := NewAgent(userID, name, apiKey)

	require.NotNil(t, agent)
	assert.NotEqual(t, agent.ID.String(), "00000000-0000-0000-0000-000000000000")
	assert.Equal(t, userID, agent.UserID)
	assert.Equal(t, name, agent.Name)
	assert.Equal(t, apiKey, agent.APIKeyEncrypted)
	assert.Nil(t, agent.LastSeenAt)
	assert.Equal(t, AgentStatusOffline, agent.Status)
	assert.False(t, agent.CreatedAt.IsZero())
}

func TestAgent_MarkOnline(t *testing.T) {
	agent := NewAgent(uuid.New(), "test", []byte("key"))
	assert.Equal(t, AgentStatusOffline, agent.Status)
	assert.Nil(t, agent.LastSeenAt)

	agent.MarkOnline()

	assert.Equal(t, AgentStatusOnline, agent.Status)
	require.NotNil(t, agent.LastSeenAt)
	assert.False(t, agent.LastSeenAt.IsZero())
}

func TestAgent_MarkOffline(t *testing.T) {
	agent := NewAgent(uuid.New(), "test", []byte("key"))
	agent.MarkOnline()
	assert.Equal(t, AgentStatusOnline, agent.Status)

	agent.MarkOffline()

	assert.Equal(t, AgentStatusOffline, agent.Status)
}

func TestAgent_IsOnline(t *testing.T) {
	agent := NewAgent(uuid.New(), "test", []byte("key"))

	assert.False(t, agent.IsOnline())

	agent.MarkOnline()
	assert.True(t, agent.IsOnline())

	agent.MarkOffline()
	assert.False(t, agent.IsOnline())
}

func TestAgent_UpdateLastSeen(t *testing.T) {
	agent := NewAgent(uuid.New(), "test", []byte("key"))
	assert.Nil(t, agent.LastSeenAt)

	agent.UpdateLastSeen()

	require.NotNil(t, agent.LastSeenAt)
	assert.False(t, agent.LastSeenAt.IsZero())
}

func TestGenerateAPIKey(t *testing.T) {
	key1, err1 := GenerateAPIKey()
	require.NoError(t, err1)

	key2, err2 := GenerateAPIKey()
	require.NoError(t, err2)

	// Keys should be 64 hex characters (32 bytes)
	assert.Len(t, key1, 64)
	assert.Len(t, key2, 64)

	// Keys should be unique
	assert.NotEqual(t, key1, key2)
}

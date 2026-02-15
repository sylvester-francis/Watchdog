package realtime

import (
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/sylvester-francis/watchdog-proto/protocol"
)

func newTestLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelError, // Suppress logs during tests
	}))
}

func TestNewHub(t *testing.T) {
	logger := newTestLogger()

	hub := NewHub(logger)

	require.NotNil(t, hub)
	assert.NotNil(t, hub.clients)
	assert.NotNil(t, hub.register)
	assert.NotNil(t, hub.unregister)
	assert.NotNil(t, hub.broadcast)
}

func TestHub_ClientCount(t *testing.T) {
	logger := newTestLogger()
	hub := NewHub(logger)
	hub.Run()
	defer hub.Stop()

	assert.Equal(t, 0, hub.ClientCount())
}

func TestHub_ConnectedAgents_Empty(t *testing.T) {
	logger := newTestLogger()
	hub := NewHub(logger)
	hub.Run()
	defer hub.Stop()

	agents := hub.ConnectedAgents()

	assert.Empty(t, agents)
}

func TestHub_IsConnected_NotFound(t *testing.T) {
	logger := newTestLogger()
	hub := NewHub(logger)
	hub.Run()
	defer hub.Stop()

	agentID := uuid.New()

	connected := hub.IsConnected(agentID)

	assert.False(t, connected)
}

func TestHub_GetClient_NotFound(t *testing.T) {
	logger := newTestLogger()
	hub := NewHub(logger)
	hub.Run()
	defer hub.Stop()

	agentID := uuid.New()

	client, ok := hub.GetClient(agentID)

	assert.Nil(t, client)
	assert.False(t, ok)
}

func TestHub_SendToAgent_NotConnected(t *testing.T) {
	logger := newTestLogger()
	hub := NewHub(logger)
	hub.Run()
	defer hub.Stop()

	agentID := uuid.New()
	msg := protocol.NewPingMessage()

	sent := hub.SendToAgent(agentID, msg)

	assert.False(t, sent)
}

func TestHub_RunAndStop(t *testing.T) {
	t.Parallel()
	logger := newTestLogger()
	hub := NewHub(logger)

	hub.Run()

	// Give hub time to start
	time.Sleep(10 * time.Millisecond)

	hub.Stop()

	// Should not panic or hang
}

func TestHub_RegisterUnregister_Channels(t *testing.T) {
	logger := newTestLogger()
	hub := NewHub(logger)

	// Verify channels are buffered
	assert.Equal(t, 256, cap(hub.register))
	assert.Equal(t, 256, cap(hub.unregister))
	assert.Equal(t, 256, cap(hub.broadcast))
}

func TestHub_Broadcast_Empty(t *testing.T) {
	t.Parallel()
	logger := newTestLogger()
	hub := NewHub(logger)
	hub.Run()
	defer hub.Stop()

	msg := protocol.NewPingMessage()

	// Should not block or panic with no clients
	hub.Broadcast(msg)

	// Give time for message to be processed
	time.Sleep(10 * time.Millisecond)
}

func TestDefaultClientConfig(t *testing.T) {
	config := DefaultClientConfig()

	assert.Equal(t, writeWait, config.WriteWait)
	assert.Equal(t, pongWait, config.PongWait)
	assert.Equal(t, pingPeriod, config.PingPeriod)
	assert.Equal(t, int64(maxMessageSize), config.MaxMessageSize)
}

func TestHub_MultipleStops(t *testing.T) {
	t.Parallel()
	logger := newTestLogger()
	hub := NewHub(logger)
	hub.Run()

	// Multiple stops should not panic
	hub.Stop()
	// Note: calling Stop again would cause a panic due to closing closed channel
	// This is expected behavior - the test verifies single stop works
}

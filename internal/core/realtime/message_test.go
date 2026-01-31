package realtime

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewMessage(t *testing.T) {
	t.Run("with payload", func(t *testing.T) {
		payload := map[string]string{"key": "value"}

		msg, err := NewMessage("test", payload)

		require.NoError(t, err)
		assert.Equal(t, "test", msg.Type)
		assert.NotNil(t, msg.Payload)
		assert.False(t, msg.Timestamp.IsZero())
	})

	t.Run("without payload", func(t *testing.T) {
		msg, err := NewMessage("test", nil)

		require.NoError(t, err)
		assert.Equal(t, "test", msg.Type)
		assert.Nil(t, msg.Payload)
	})

	t.Run("with unmarshalable payload", func(t *testing.T) {
		// Channels can't be marshaled to JSON
		payload := make(chan int)

		_, err := NewMessage("test", payload)

		assert.Error(t, err)
	})
}

func TestMessage_ParsePayload(t *testing.T) {
	t.Run("valid payload", func(t *testing.T) {
		payload := AuthPayload{APIKey: "test-key", Version: "1.0"}
		msg, _ := NewMessage(MsgTypeAuth, payload)

		var parsed AuthPayload
		err := msg.ParsePayload(&parsed)

		require.NoError(t, err)
		assert.Equal(t, "test-key", parsed.APIKey)
		assert.Equal(t, "1.0", parsed.Version)
	})

	t.Run("nil payload", func(t *testing.T) {
		msg, _ := NewMessage(MsgTypePing, nil)

		var parsed AuthPayload
		err := msg.ParsePayload(&parsed)

		require.NoError(t, err)
		assert.Empty(t, parsed.APIKey)
	})

	t.Run("invalid payload type", func(t *testing.T) {
		msg := &Message{
			Type:    "test",
			Payload: json.RawMessage(`"not an object"`),
		}

		var parsed AuthPayload
		err := msg.ParsePayload(&parsed)

		assert.Error(t, err)
	})
}

func TestMustNewMessage(t *testing.T) {
	t.Run("valid payload", func(t *testing.T) {
		payload := map[string]string{"key": "value"}

		msg := MustNewMessage("test", payload)

		assert.NotNil(t, msg)
		assert.Equal(t, "test", msg.Type)
	})

	t.Run("invalid payload panics", func(t *testing.T) {
		assert.Panics(t, func() {
			MustNewMessage("test", make(chan int))
		})
	})
}

func TestNewAuthMessage(t *testing.T) {
	apiKey := "test-api-key"
	version := "1.0.0"

	msg := NewAuthMessage(apiKey, version)

	assert.Equal(t, MsgTypeAuth, msg.Type)
	assert.False(t, msg.Timestamp.IsZero())

	var payload AuthPayload
	require.NoError(t, msg.ParsePayload(&payload))
	assert.Equal(t, apiKey, payload.APIKey)
	assert.Equal(t, version, payload.Version)
}

func TestNewAuthAckMessage(t *testing.T) {
	agentID := "agent-123"
	agentName := "test-agent"

	msg := NewAuthAckMessage(agentID, agentName)

	assert.Equal(t, MsgTypeAuthAck, msg.Type)

	var payload AuthAckPayload
	require.NoError(t, msg.ParsePayload(&payload))
	assert.Equal(t, agentID, payload.AgentID)
	assert.Equal(t, agentName, payload.AgentName)
}

func TestNewAuthErrorMessage(t *testing.T) {
	errorMsg := "invalid API key"

	msg := NewAuthErrorMessage(errorMsg)

	assert.Equal(t, MsgTypeAuthError, msg.Type)

	var payload AuthErrorPayload
	require.NoError(t, msg.ParsePayload(&payload))
	assert.Equal(t, errorMsg, payload.Error)
}

func TestNewTaskMessage(t *testing.T) {
	monitorID := "mon-123"
	monitorType := "http"
	target := "https://example.com"
	interval := 30
	timeout := 10

	msg := NewTaskMessage(monitorID, monitorType, target, interval, timeout)

	assert.Equal(t, MsgTypeTask, msg.Type)

	var payload TaskPayload
	require.NoError(t, msg.ParsePayload(&payload))
	assert.Equal(t, monitorID, payload.MonitorID)
	assert.Equal(t, monitorType, payload.Type)
	assert.Equal(t, target, payload.Target)
	assert.Equal(t, interval, payload.Interval)
	assert.Equal(t, timeout, payload.Timeout)
}

func TestNewHeartbeatMessage(t *testing.T) {
	monitorID := "mon-123"
	status := "up"
	latencyMs := 42
	errorMsg := ""

	msg := NewHeartbeatMessage(monitorID, status, latencyMs, errorMsg)

	assert.Equal(t, MsgTypeHeartbeat, msg.Type)

	var payload HeartbeatPayload
	require.NoError(t, msg.ParsePayload(&payload))
	assert.Equal(t, monitorID, payload.MonitorID)
	assert.Equal(t, status, payload.Status)
	assert.Equal(t, latencyMs, payload.LatencyMs)
	assert.Equal(t, errorMsg, payload.ErrorMessage)
}

func TestNewPingPongMessages(t *testing.T) {
	t.Run("ping", func(t *testing.T) {
		msg := NewPingMessage()
		assert.Equal(t, MsgTypePing, msg.Type)
	})

	t.Run("pong", func(t *testing.T) {
		msg := NewPongMessage()
		assert.Equal(t, MsgTypePong, msg.Type)
	})
}

func TestNewErrorMessage(t *testing.T) {
	code := "ERR_001"
	message := "Something went wrong"

	msg := NewErrorMessage(code, message)

	assert.Equal(t, MsgTypeError, msg.Type)

	var payload ErrorPayload
	require.NoError(t, msg.ParsePayload(&payload))
	assert.Equal(t, code, payload.Code)
	assert.Equal(t, message, payload.Message)
}

func TestMessage_JSONSerialization(t *testing.T) {
	original := NewAuthMessage("api-key", "1.0")

	data, err := json.Marshal(original)
	require.NoError(t, err)

	var decoded Message
	err = json.Unmarshal(data, &decoded)
	require.NoError(t, err)

	assert.Equal(t, original.Type, decoded.Type)
	assert.Equal(t, string(original.Payload), string(decoded.Payload))
	assert.WithinDuration(t, original.Timestamp, decoded.Timestamp, time.Second)
}

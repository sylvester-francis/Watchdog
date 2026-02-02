package main

import (
	"encoding/json"
	"time"
)

// Message types for WebSocket communication.
const (
	MsgTypeAuth      = "auth"
	MsgTypeAuthAck   = "auth_ack"
	MsgTypeAuthError = "auth_error"
	MsgTypeTask      = "task"
	MsgTypeHeartbeat = "heartbeat"
	MsgTypePing      = "ping"
	MsgTypePong      = "pong"
	MsgTypeError     = "error"
)

// Message represents a WebSocket message envelope.
type Message struct {
	Type      string          `json:"type"`
	Payload   json.RawMessage `json:"payload,omitempty"`
	Timestamp time.Time       `json:"timestamp"`
}

// NewMessage creates a new message with the current timestamp.
func NewMessage(msgType string, payload any) *Message {
	var rawPayload json.RawMessage
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			// Payload types are known at compile time and should always be marshalable.
			// A failure here indicates a programming error.
			panic("failed to marshal message payload: " + err.Error())
		}
		rawPayload = data
	}

	return &Message{
		Type:      msgType,
		Payload:   rawPayload,
		Timestamp: time.Now(),
	}
}

// ParsePayload unmarshals the payload into the provided type.
func (m *Message) ParsePayload(v any) error {
	if m.Payload == nil {
		return nil
	}
	return json.Unmarshal(m.Payload, v)
}

// AuthPayload is sent by agent to authenticate.
type AuthPayload struct {
	APIKey  string `json:"api_key"`
	Version string `json:"version,omitempty"`
}

// AuthAckPayload is sent by hub to confirm authentication.
type AuthAckPayload struct {
	AgentID   string `json:"agent_id"`
	AgentName string `json:"agent_name"`
}

// AuthErrorPayload is sent by hub when authentication fails.
type AuthErrorPayload struct {
	Error string `json:"error"`
}

// TaskPayload describes a monitoring task for the agent.
type TaskPayload struct {
	MonitorID string `json:"monitor_id"`
	Type      string `json:"type"`
	Target    string `json:"target"`
	Interval  int    `json:"interval"`
	Timeout   int    `json:"timeout"`
}

// HeartbeatPayload is sent by agent with check results.
type HeartbeatPayload struct {
	MonitorID    string `json:"monitor_id"`
	Status       string `json:"status"`
	LatencyMs    int    `json:"latency_ms,omitempty"`
	ErrorMessage string `json:"error_message,omitempty"`
}

// NewAuthMessage creates an authentication message.
func NewAuthMessage(apiKey, version string) *Message {
	return NewMessage(MsgTypeAuth, AuthPayload{
		APIKey:  apiKey,
		Version: version,
	})
}

// NewHeartbeatMessage creates a heartbeat message.
func NewHeartbeatMessage(monitorID, status string, latencyMs int, errorMsg string) *Message {
	return NewMessage(MsgTypeHeartbeat, HeartbeatPayload{
		MonitorID:    monitorID,
		Status:       status,
		LatencyMs:    latencyMs,
		ErrorMessage: errorMsg,
	})
}

// NewPongMessage creates a pong message.
func NewPongMessage() *Message {
	return NewMessage(MsgTypePong, nil)
}

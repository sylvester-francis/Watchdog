package realtime

import (
	"encoding/json"
	"log/slog"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
	"github.com/sylvester-francis/watchdog-proto/protocol"
)

// maxHeartbeatsPerWindow caps heartbeats per rate-limit window (H-009).
const maxHeartbeatsPerWindow = 200

// Client configuration constants.
const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer (H-004: 64 KB cap).
	maxMessageSize = 64 * 1024 // 64 KB
)

// HeartbeatCallback is called when a heartbeat is received from an agent.
type HeartbeatCallback func(agentID uuid.UUID, payload *protocol.HeartbeatPayload)

// Client represents a connected agent.
type Client struct {
	AgentID     uuid.UUID
	AgentName   string
	conn        *websocket.Conn
	send        chan *protocol.Message
	hub         *Hub
	logger      *slog.Logger
	closeOnce   sync.Once
	closeCh     chan struct{}
	onHeartbeat HeartbeatCallback
	hbCount     atomic.Int64 // heartbeats in current window (H-009)
}

// NewClient creates a new client for the given connection.
func NewClient(hub *Hub, conn *websocket.Conn, agentID uuid.UUID, agentName string, logger *slog.Logger) *Client {
	return &Client{
		AgentID:   agentID,
		AgentName: agentName,
		conn:      conn,
		send:      make(chan *protocol.Message, 256),
		hub:       hub,
		logger:    logger,
		closeCh:   make(chan struct{}),
	}
}

// SetHeartbeatCallback sets the callback for heartbeat processing.
func (c *Client) SetHeartbeatCallback(cb HeartbeatCallback) {
	c.onHeartbeat = cb
}

// Start begins the read and write pumps for this client.
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

// Send queues a message to be sent to the client.
// Returns false if the send buffer is full or client is closed.
func (c *Client) Send(message *protocol.Message) bool {
	select {
	case c.send <- message:
		return true
	case <-c.closeCh:
		return false
	default:
		c.logger.Warn("send buffer full, dropping message",
			slog.String("agent_id", c.AgentID.String()),
			slog.String("message_type", message.Type),
		)
		return false
	}
}

// Close closes the client connection.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		close(c.closeCh)
		c.conn.Close()
	})
}

// IsClosed returns true if the client connection is closed.
func (c *Client) IsClosed() bool {
	select {
	case <-c.closeCh:
		return true
	default:
		return false
	}
}

// CloseCh returns the close channel for waiting on disconnection.
func (c *Client) CloseCh() <-chan struct{} {
	return c.closeCh
}

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	if err := c.conn.SetReadDeadline(time.Now().Add(pongWait)); err != nil {
		c.logger.Error("failed to set read deadline", slog.String("error", err.Error()))
		return
	}
	c.conn.SetPongHandler(func(string) error {
		return c.conn.SetReadDeadline(time.Now().Add(pongWait))
	})

	for {
		_, data, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				c.logger.Error("websocket read error",
					slog.String("agent_id", c.AgentID.String()),
					slog.String("error", err.Error()),
				)
			}
			return
		}

		var msg protocol.Message
		if err := json.Unmarshal(data, &msg); err != nil {
			c.logger.Warn("failed to parse message",
				slog.String("agent_id", c.AgentID.String()),
				slog.String("error", err.Error()),
			)
			continue
		}

		c.handleMessage(&msg)
	}
}

// writePump pumps messages from the send channel to the websocket connection.
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				c.logger.Error("failed to set write deadline", slog.String("error", err.Error()))
				return
			}
			if !ok {
				//nolint:errcheck // Best-effort close message, connection is closing anyway.
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			data, err := json.Marshal(message)
			if err != nil {
				c.logger.Error("failed to marshal message",
					slog.String("agent_id", c.AgentID.String()),
					slog.String("error", err.Error()),
				)
				continue
			}

			if err := c.conn.WriteMessage(websocket.TextMessage, data); err != nil {
				c.logger.Error("failed to write message",
					slog.String("agent_id", c.AgentID.String()),
					slog.String("error", err.Error()),
				)
				return
			}

		case <-ticker.C:
			// H-009: reset heartbeat rate limiter each ping period (~54s).
			c.hbCount.Store(0)

			if err := c.conn.SetWriteDeadline(time.Now().Add(writeWait)); err != nil {
				return
			}
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.closeCh:
			return
		}
	}
}

// handleMessage processes incoming messages from the agent.
func (c *Client) handleMessage(msg *protocol.Message) {
	switch msg.Type {
	case protocol.MsgTypeHeartbeat:
		c.handleHeartbeat(msg)
	case protocol.MsgTypePong:
		// Pong received, connection is alive
		c.logger.Debug("pong received", slog.String("agent_id", c.AgentID.String()))
	default:
		c.logger.Warn("unknown message type",
			slog.String("agent_id", c.AgentID.String()),
			slog.String("type", msg.Type),
		)
	}
}

// handleHeartbeat processes heartbeat messages from the agent.
func (c *Client) handleHeartbeat(msg *protocol.Message) {
	// H-009: cap heartbeats per rate-limit window to prevent DB insert storms.
	if c.hbCount.Add(1) > int64(maxHeartbeatsPerWindow) {
		c.logger.Warn("heartbeat rate limit exceeded, dropping",
			slog.String("agent_id", c.AgentID.String()),
		)
		return
	}

	var payload protocol.HeartbeatPayload
	if err := msg.ParsePayload(&payload); err != nil {
		c.logger.Warn("failed to parse heartbeat payload",
			slog.String("agent_id", c.AgentID.String()),
			slog.String("error", err.Error()),
		)
		return
	}

	c.logger.Debug("heartbeat received",
		slog.String("agent_id", c.AgentID.String()),
		slog.String("monitor_id", payload.MonitorID),
		slog.String("status", payload.Status),
		slog.Int("latency_ms", payload.LatencyMs),
	)

	if c.onHeartbeat != nil {
		c.onHeartbeat(c.AgentID, &payload)
	}
}

// MessageHandler is a callback for handling messages.
type MessageHandler func(client *Client, msg *protocol.Message)

// ClientConfig holds client configuration.
type ClientConfig struct {
	WriteWait      time.Duration
	PongWait       time.Duration
	PingPeriod     time.Duration
	MaxMessageSize int64
}

// DefaultClientConfig returns the default client configuration.
func DefaultClientConfig() ClientConfig {
	return ClientConfig{
		WriteWait:      writeWait,
		PongWait:       pongWait,
		PingPeriod:     pingPeriod,
		MaxMessageSize: maxMessageSize,
	}
}

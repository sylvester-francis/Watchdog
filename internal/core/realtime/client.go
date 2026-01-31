package realtime

import (
	"encoding/json"
	"log/slog"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

// Client configuration constants.
const (
	// Time allowed to write a message to the peer.
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer.
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait.
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer.
	maxMessageSize = 512 * 1024 // 512 KB
)

// Client represents a connected agent.
type Client struct {
	AgentID   uuid.UUID
	AgentName string
	conn      *websocket.Conn
	send      chan *Message
	hub       *Hub
	logger    *slog.Logger
	closeOnce sync.Once
	closeCh   chan struct{}
}

// NewClient creates a new client for the given connection.
func NewClient(hub *Hub, conn *websocket.Conn, agentID uuid.UUID, agentName string, logger *slog.Logger) *Client {
	return &Client{
		AgentID:   agentID,
		AgentName: agentName,
		conn:      conn,
		send:      make(chan *Message, 256),
		hub:       hub,
		logger:    logger,
		closeCh:   make(chan struct{}),
	}
}

// Start begins the read and write pumps for this client.
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

// Send queues a message to be sent to the client.
// Returns false if the send buffer is full or client is closed.
func (c *Client) Send(message *Message) bool {
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

// readPump pumps messages from the websocket connection to the hub.
func (c *Client) readPump() {
	defer func() {
		c.hub.Unregister(c)
		c.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
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

		var msg Message
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
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
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
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}

		case <-c.closeCh:
			return
		}
	}
}

// handleMessage processes incoming messages from the agent.
func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case MsgTypeHeartbeat:
		c.handleHeartbeat(msg)
	case MsgTypePong:
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
func (c *Client) handleHeartbeat(msg *Message) {
	var payload HeartbeatPayload
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

	// TODO: Process heartbeat through monitor service
}

// MessageHandler is a callback for handling messages.
type MessageHandler func(client *Client, msg *Message)

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

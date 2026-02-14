package realtime

import (
	"log/slog"
	"sync"

	"github.com/google/uuid"
	"github.com/sylvester-francis/watchdog-proto/protocol"
)

// Hub maintains the set of active agent connections and broadcasts messages.
type Hub struct {
	clients    map[uuid.UUID]*Client
	register   chan *Client
	unregister chan *Client
	broadcast  chan *protocol.Message
	mu         sync.RWMutex
	logger     *slog.Logger
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewHub creates a new Hub instance.
func NewHub(logger *slog.Logger) *Hub {
	return &Hub{
		clients:    make(map[uuid.UUID]*Client),
		register:   make(chan *Client, 256),
		unregister: make(chan *Client, 256),
		broadcast:  make(chan *protocol.Message, 256),
		logger:     logger,
		stopCh:     make(chan struct{}),
	}
}

// Run starts the hub's main event loop.
func (h *Hub) Run() {
	h.wg.Add(1)
	go func() {
		defer h.wg.Done()
		h.run()
	}()
}

func (h *Hub) run() {
	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case <-h.stopCh:
			h.closeAllClients()
			return
		}
	}
}

// Stop gracefully stops the hub.
func (h *Hub) Stop() {
	close(h.stopCh)
	h.wg.Wait()
}

// Register adds a client to the hub.
func (h *Hub) Register(client *Client) {
	h.register <- client
}

// Unregister removes a client from the hub.
func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(message *protocol.Message) {
	h.broadcast <- message
}

// SendToAgent sends a message to a specific agent.
func (h *Hub) SendToAgent(agentID uuid.UUID, message *protocol.Message) bool {
	h.mu.RLock()
	client, ok := h.clients[agentID]
	h.mu.RUnlock()

	if !ok {
		return false
	}

	return client.Send(message)
}

// GetClient returns a client by agent ID.
func (h *Hub) GetClient(agentID uuid.UUID) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	client, ok := h.clients[agentID]
	return client, ok
}

// IsConnected checks if an agent is connected.
func (h *Hub) IsConnected(agentID uuid.UUID) bool {
	h.mu.RLock()
	defer h.mu.RUnlock()
	_, ok := h.clients[agentID]
	return ok
}

// ConnectedAgents returns a list of connected agent IDs.
func (h *Hub) ConnectedAgents() []uuid.UUID {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ids := make([]uuid.UUID, 0, len(h.clients))
	for id := range h.clients {
		ids = append(ids, id)
	}
	return ids
}

// ClientCount returns the number of connected clients.
func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) registerClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	// Close existing client with same ID if exists
	if existing, ok := h.clients[client.AgentID]; ok {
		existing.Close()
		h.logger.Warn("replaced existing client",
			slog.String("agent_id", client.AgentID.String()),
		)
	}

	h.clients[client.AgentID] = client
	h.logger.Info("client registered",
		slog.String("agent_id", client.AgentID.String()),
		slog.Int("total_clients", len(h.clients)),
	)
}

func (h *Hub) unregisterClient(client *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()

	if existing, ok := h.clients[client.AgentID]; ok && existing == client {
		delete(h.clients, client.AgentID)
		client.Close()
		h.logger.Info("client unregistered",
			slog.String("agent_id", client.AgentID.String()),
			slog.Int("total_clients", len(h.clients)),
		)
	}
}

func (h *Hub) broadcastMessage(message *protocol.Message) {
	h.mu.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for _, client := range h.clients {
		clients = append(clients, client)
	}
	h.mu.RUnlock()

	for _, client := range clients {
		client.Send(message)
	}
}

func (h *Hub) closeAllClients() {
	h.mu.Lock()
	defer h.mu.Unlock()

	for _, client := range h.clients {
		client.Close()
	}
	h.clients = make(map[uuid.UUID]*Client)
	h.logger.Info("all clients closed")
}

package websocket

import (
	"sync"

	"github.com/sirupsen/logrus"
)

// Hub maintains active WebSocket connections and broadcasts messages
type Hub struct {
	// Registered clients
	clients map[*Client]bool

	// Inbound messages from clients
	broadcast chan []byte

	// Register requests from clients
	register chan *Client

	// Unregister requests from clients
	unregister chan *Client

	// Mutex for thread safety
	mutex sync.RWMutex

	// Stop channel
	stop chan struct{}
}

// NewHub creates a new WebSocket hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		stop:       make(chan struct{}),
	}
}

// Run starts the hub and handles client connections
func (h *Hub) Run() {
	logrus.Info("Starting WebSocket hub")

	for {
		select {
		case client := <-h.register:
			h.registerClient(client)

		case client := <-h.unregister:
			h.unregisterClient(client)

		case message := <-h.broadcast:
			h.broadcastMessage(message)

		case <-h.stop:
			logrus.Info("Stopping WebSocket hub")
			h.closeAllClients()
			return
		}
	}
}

// Stop stops the hub
func (h *Hub) Stop() {
	close(h.stop)
}

// RegisterClient registers a new client
func (h *Hub) RegisterClient(client *Client) {
	h.register <- client
}

// UnregisterClient unregisters a client
func (h *Hub) UnregisterClient(client *Client) {
	h.unregister <- client
}

// Broadcast sends a message to all connected clients
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}

// GetClientCount returns the number of connected clients
func (h *Hub) GetClientCount() int {
	h.mutex.RLock()
	defer h.mutex.RUnlock()
	return len(h.clients)
}

// registerClient handles client registration
func (h *Hub) registerClient(client *Client) {
	h.mutex.Lock()
	h.clients[client] = true
	h.mutex.Unlock()

	logrus.WithField("client_count", len(h.clients)).Debug("Client connected")

	// Send welcome message
	welcomeMsg := []byte(`{"type":"connected","message":"Connected to CasDash WebSocket"}`)
	select {
	case client.send <- welcomeMsg:
	default:
		h.unregisterClient(client)
	}
}

// unregisterClient handles client disconnection
func (h *Hub) unregisterClient(client *Client) {
	h.mutex.Lock()
	if _, ok := h.clients[client]; ok {
		delete(h.clients, client)
		close(client.send)
	}
	h.mutex.Unlock()

	logrus.WithField("client_count", len(h.clients)).Debug("Client disconnected")
}

// broadcastMessage sends a message to all connected clients
func (h *Hub) broadcastMessage(message []byte) {
	h.mutex.RLock()
	clients := make([]*Client, 0, len(h.clients))
	for client := range h.clients {
		clients = append(clients, client)
	}
	h.mutex.RUnlock()

	for _, client := range clients {
		select {
		case client.send <- message:
		default:
			h.unregisterClient(client)
		}
	}
}

// closeAllClients closes all client connections
func (h *Hub) closeAllClients() {
	h.mutex.Lock()
	for client := range h.clients {
		close(client.send)
	}
	h.clients = make(map[*Client]bool)
	h.mutex.Unlock()
}

// BroadcastToUser sends a message to a specific user's connections
func (h *Hub) BroadcastToUser(userID int, message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		if client.userID == userID {
			select {
			case client.send <- message:
			default:
				go h.UnregisterClient(client)
			}
		}
	}
}

// BroadcastToChannel sends a message to clients subscribed to a specific channel
func (h *Hub) BroadcastToChannel(channel string, message []byte) {
	h.mutex.RLock()
	defer h.mutex.RUnlock()

	for client := range h.clients {
		if client.isSubscribedToChannel(channel) {
			select {
			case client.send <- message:
			default:
				go h.UnregisterClient(client)
			}
		}
	}
}
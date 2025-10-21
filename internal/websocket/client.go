package websocket

import (
	"bytes"
	"encoding/json"
	"net/http"
	"time"

	"github.com/gorilla/websocket"
	"github.com/sirupsen/logrus"
)

const (
	// Time allowed to write a message to the peer
	writeWait = 10 * time.Second

	// Time allowed to read the next pong message from the peer
	pongWait = 60 * time.Second

	// Send pings to peer with this period. Must be less than pongWait
	pingPeriod = (pongWait * 9) / 10

	// Maximum message size allowed from peer
	maxMessageSize = 512
)

var (
	newline = []byte{'\n'}
	space   = []byte{' '}
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		// Allow connections from any origin in development
		// TODO: Implement proper origin checking for production
		return true
	},
}

// Client represents a WebSocket client connection
type Client struct {
	hub    *Hub
	conn   *websocket.Conn
	send   chan []byte
	userID int

	// Subscriptions
	channels map[string]bool
}

// Message represents a WebSocket message
type Message struct {
	Type    string      `json:"type"`
	Channel string      `json:"channel,omitempty"`
	Payload interface{} `json:"payload,omitempty"`
}

// NewClient creates a new WebSocket client
func NewClient(hub *Hub, conn *websocket.Conn, userID int) *Client {
	return &Client{
		hub:      hub,
		conn:     conn,
		send:     make(chan []byte, 256),
		userID:   userID,
		channels: make(map[string]bool),
	}
}

// Start starts the client's read and write pumps
func (c *Client) Start() {
	go c.writePump()
	go c.readPump()
}

// readPump pumps messages from the websocket connection to the hub
func (c *Client) readPump() {
	defer func() {
		c.hub.UnregisterClient(c)
		c.conn.Close()
	}()

	c.conn.SetReadLimit(maxMessageSize)
	c.conn.SetReadDeadline(time.Now().Add(pongWait))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(pongWait))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				logrus.WithError(err).Error("WebSocket error")
			}
			break
		}

		message = bytes.TrimSpace(bytes.Replace(message, newline, space, -1))
		c.handleMessage(message)
	}
}

// writePump pumps messages from the hub to the websocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(pingPeriod)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if !ok {
				// The hub closed the channel
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			// Add queued chat messages to the current websocket message
			n := len(c.send)
			for i := 0; i < n; i++ {
				w.Write(newline)
				w.Write(<-c.send)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(writeWait))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage processes incoming messages from the client
func (c *Client) handleMessage(data []byte) {
	var msg Message
	if err := json.Unmarshal(data, &msg); err != nil {
		logrus.WithError(err).Error("Failed to unmarshal WebSocket message")
		return
	}

	switch msg.Type {
	case "subscribe":
		c.subscribe(msg.Channel)
	case "unsubscribe":
		c.unsubscribe(msg.Channel)
	case "ping":
		c.sendPong()
	default:
		logrus.WithField("type", msg.Type).Debug("Unknown WebSocket message type")
	}
}

// subscribe subscribes the client to a channel
func (c *Client) subscribe(channel string) {
	if channel == "" {
		return
	}

	c.channels[channel] = true
	logrus.WithFields(logrus.Fields{
		"user_id": c.userID,
		"channel": channel,
	}).Debug("Client subscribed to channel")

	// Send subscription confirmation
	response := Message{
		Type:    "subscribed",
		Channel: channel,
		Payload: map[string]string{"status": "success"},
	}
	c.sendMessage(response)
}

// unsubscribe unsubscribes the client from a channel
func (c *Client) unsubscribe(channel string) {
	if channel == "" {
		return
	}

	delete(c.channels, channel)
	logrus.WithFields(logrus.Fields{
		"user_id": c.userID,
		"channel": channel,
	}).Debug("Client unsubscribed from channel")

	// Send unsubscription confirmation
	response := Message{
		Type:    "unsubscribed",
		Channel: channel,
		Payload: map[string]string{"status": "success"},
	}
	c.sendMessage(response)
}

// sendPong sends a pong response
func (c *Client) sendPong() {
	response := Message{
		Type:    "pong",
		Payload: map[string]interface{}{"timestamp": time.Now().Unix()},
	}
	c.sendMessage(response)
}

// sendMessage sends a message to the client
func (c *Client) sendMessage(msg Message) {
	data, err := json.Marshal(msg)
	if err != nil {
		logrus.WithError(err).Error("Failed to marshal WebSocket message")
		return
	}

	select {
	case c.send <- data:
	default:
		c.hub.UnregisterClient(c)
	}
}

// isSubscribedToChannel checks if the client is subscribed to a channel
func (c *Client) isSubscribedToChannel(channel string) bool {
	return c.channels[channel]
}

// ServeWS handles websocket requests from the peer
func ServeWS(hub *Hub, w http.ResponseWriter, r *http.Request, userID int) {
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		logrus.WithError(err).Error("Failed to upgrade WebSocket connection")
		return
	}

	client := NewClient(hub, conn, userID)
	hub.RegisterClient(client)

	// Start the client pumps
	client.Start()
}
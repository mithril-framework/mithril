package websocket

import (
	"context"
	"encoding/json"
	"log"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/websocket/v2"
)

// Server manages WebSocket connections
type Server struct {
	clients    map[*Client]bool
	rooms      map[string]map[*Client]bool
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
	middleware []func(*Client) error
}

// Client represents a WebSocket client
type Client struct {
	ID       string
	Conn     *websocket.Conn
	Server   *Server
	Send     chan []byte
	Rooms    map[string]bool
	Metadata map[string]interface{}
	mu       sync.RWMutex
}

// Message represents a WebSocket message
type Message struct {
	Type    string                 `json:"type"`
	Room    string                 `json:"room,omitempty"`
	Data    interface{}            `json:"data"`
	From    string                 `json:"from,omitempty"`
	To      string                 `json:"to,omitempty"`
	Meta    map[string]interface{} `json:"meta,omitempty"`
	Timestamp time.Time            `json:"timestamp"`
}

// NewServer creates a new WebSocket server
func NewServer() *Server {
	return &Server{
		clients:    make(map[*Client]bool),
		rooms:      make(map[string]map[*Client]bool),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		middleware: make([]func(*Client) error, 0),
	}
}

// Run starts the WebSocket server
func (s *Server) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-s.register:
			s.registerClient(client)
		case client := <-s.unregister:
			s.unregisterClient(client)
		case message := <-s.broadcast:
			s.broadcastMessage(message)
		}
	}
}

// Use adds middleware to the WebSocket server
func (s *Server) Use(middleware func(*Client) error) {
	s.middleware = append(s.middleware, middleware)
}

// HandleConnection creates a Fiber WebSocket handler
func (s *Server) HandleConnection(c *websocket.Conn) {
	client := &Client{
		ID:       generateClientID(),
		Conn:     c,
		Server:   s,
		Send:     make(chan []byte, 256),
		Rooms:    make(map[string]bool),
		Metadata: make(map[string]interface{}),
	}

	// Run middleware
	for _, mw := range s.middleware {
		if err := mw(client); err != nil {
			log.Printf("[WebSocket] Middleware rejected client: %v", err)
			c.Close()
			return
		}
	}

	s.register <- client

	// Start goroutines for reading and writing
	go client.writePump()
	go client.readPump()
}

// Upgrade creates a Fiber middleware to upgrade HTTP to WebSocket
func (s *Server) Upgrade() fiber.Handler {
	return websocket.New(func(c *websocket.Conn) {
		s.HandleConnection(c)
	})
}

// Broadcast sends a message to all connected clients
func (s *Server) Broadcast(message *Message) {
	message.Timestamp = time.Now()
	s.broadcast <- message
}

// BroadcastToRoom sends a message to all clients in a room
func (s *Server) BroadcastToRoom(room string, message *Message) {
	message.Room = room
	message.Timestamp = time.Now()
	s.broadcast <- message
}

// SendToClient sends a message to a specific client
func (s *Server) SendToClient(clientID string, message *Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for client := range s.clients {
		if client.ID == clientID {
			data, _ := json.Marshal(message)
			select {
			case client.Send <- data:
			default:
				close(client.Send)
				delete(s.clients, client)
			}
			break
		}
	}
}

// GetClients returns all connected clients
func (s *Server) GetClients() []*Client {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]*Client, 0, len(s.clients))
	for client := range s.clients {
		clients = append(clients, client)
	}
	return clients
}

// GetClientCount returns the number of connected clients
func (s *Server) GetClientCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.clients)
}

// GetRoomClients returns all clients in a specific room
func (s *Server) GetRoomClients(room string) []*Client {
	s.mu.RLock()
	defer s.mu.RUnlock()

	clients := make([]*Client, 0)
	if roomClients, ok := s.rooms[room]; ok {
		for client := range roomClients {
			clients = append(clients, client)
		}
	}
	return clients
}

// GetRoomCount returns the number of rooms
func (s *Server) GetRoomCount() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.rooms)
}

func (s *Server) registerClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.clients[client] = true
	log.Printf("[WebSocket] Client connected: %s (Total: %d)", client.ID, len(s.clients))
}

func (s *Server) unregisterClient(client *Client) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.clients[client]; ok {
		// Remove from all rooms
		for room := range client.Rooms {
			if roomClients, exists := s.rooms[room]; exists {
				delete(roomClients, client)
				if len(roomClients) == 0 {
					delete(s.rooms, room)
				}
			}
		}

		delete(s.clients, client)
		close(client.Send)
		log.Printf("[WebSocket] Client disconnected: %s (Total: %d)", client.ID, len(s.clients))
	}
}

func (s *Server) broadcastMessage(message *Message) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("[WebSocket] Failed to marshal message: %v", err)
		return
	}

	// If room is specified, broadcast to room only
	if message.Room != "" {
		if roomClients, ok := s.rooms[message.Room]; ok {
			for client := range roomClients {
				select {
				case client.Send <- data:
				default:
					close(client.Send)
					delete(s.clients, client)
				}
			}
		}
		return
	}

	// If recipient is specified, send to that client only
	if message.To != "" {
		for client := range s.clients {
			if client.ID == message.To {
				select {
				case client.Send <- data:
				default:
					close(client.Send)
					delete(s.clients, client)
				}
				return
			}
		}
		return
	}

	// Otherwise broadcast to all clients
	for client := range s.clients {
		select {
		case client.Send <- data:
		default:
			close(client.Send)
			delete(s.clients, client)
		}
	}
}

// JoinRoom adds the client to a room
func (c *Client) JoinRoom(room string) {
	c.Server.mu.Lock()
	defer c.Server.mu.Unlock()

	if c.Server.rooms[room] == nil {
		c.Server.rooms[room] = make(map[*Client]bool)
	}
	c.Server.rooms[room][c] = true

	c.mu.Lock()
	c.Rooms[room] = true
	c.mu.Unlock()

	log.Printf("[WebSocket] Client %s joined room: %s", c.ID, room)
}

// LeaveRoom removes the client from a room
func (c *Client) LeaveRoom(room string) {
	c.Server.mu.Lock()
	defer c.Server.mu.Unlock()

	if roomClients, ok := c.Server.rooms[room]; ok {
		delete(roomClients, c)
		if len(roomClients) == 0 {
			delete(c.Server.rooms, room)
		}
	}

	c.mu.Lock()
	delete(c.Rooms, room)
	c.mu.Unlock()

	log.Printf("[WebSocket] Client %s left room: %s", c.ID, room)
}

// Send sends a message to the client
func (c *Client) SendMessage(message *Message) {
	message.Timestamp = time.Now()
	data, err := json.Marshal(message)
	if err != nil {
		log.Printf("[WebSocket] Failed to marshal message: %v", err)
		return
	}

	select {
	case c.Send <- data:
	default:
		log.Printf("[WebSocket] Client %s send buffer full", c.ID)
	}
}

// readPump reads messages from the WebSocket connection
func (c *Client) readPump() {
	defer func() {
		c.Server.unregister <- c
		c.Conn.Close()
	}()

	_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		_ = c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		var message Message
		err := c.Conn.ReadJSON(&message)
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				log.Printf("[WebSocket] Error: %v", err)
			}
			break
		}

		message.From = c.ID
		message.Timestamp = time.Now()

		// Handle special message types
		switch message.Type {
		case "join":
			if room, ok := message.Data.(string); ok {
				c.JoinRoom(room)
			}
		case "leave":
			if room, ok := message.Data.(string); ok {
				c.LeaveRoom(room)
			}
		case "ping":
			c.SendMessage(&Message{Type: "pong", Data: "pong"})
		default:
			// Broadcast the message
			c.Server.broadcast <- &message
		}
	}
}

// writePump writes messages to the WebSocket connection
func (c *Client) writePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}

		case <-ticker.C:
			_ = c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func generateClientID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}


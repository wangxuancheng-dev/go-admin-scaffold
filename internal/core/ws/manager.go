package ws

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"go-admin-scaffold/pkg/logger"

	"github.com/coder/websocket"
)

// MessageType defines the type of WebSocket message
const (
	MessageTypePrivate      = 1
	MessageTypeGroup        = 2
	MessageTypeAnnouncement = 3
)

// Message represents a WebSocket message
type Message struct {
	Type      int    `json:"type"`
	From      string `json:"from"`
	To        string `json:"to"` // User ID or Group ID
	Content   string `json:"content"`
	Timestamp int64  `json:"timestamp"`
}

// Client represents a WebSocket client
type Client struct {
	ID      string
	Conn    *websocket.Conn
	Send    chan []byte
	Manager *Manager
	Groups  map[string]bool
}

// Manager manages WebSocket connections and message broadcasting
type Manager struct {
	Clients    map[string]*Client
	Groups     map[string]map[string]bool // group -> userIDs
	Register   chan *Client
	Unregister chan *Client
	Broadcast  chan *Message
	mu         sync.RWMutex
}

// NewManager creates a new WebSocket manager
func NewManager() *Manager {
	return &Manager{
		Clients:    make(map[string]*Client),
		Groups:     make(map[string]map[string]bool),
		Register:   make(chan *Client),
		Unregister: make(chan *Client),
		Broadcast:  make(chan *Message),
	}
}

// Start starts the WebSocket manager
func (m *Manager) Start() {
	ctx := context.Background()
	for {
		select {
		case client := <-m.Register:
			m.mu.Lock()
			m.Clients[client.ID] = client
			m.mu.Unlock()
			logger.Debug(ctx, "ws client registered", "client_id", client.ID)

		case client := <-m.Unregister:
			if _, ok := m.Clients[client.ID]; ok {
				m.mu.Lock()
				for groupID := range client.Groups {
					if group, exists := m.Groups[groupID]; exists {
						delete(group, client.ID)

						if len(group) == 0 {
							delete(m.Groups, groupID)
						} else {
							notifyMsg := &Message{
								Type:      MessageTypeAnnouncement,
								From:      client.ID,
								Content:   fmt.Sprintf("用户 %s 离开了群组（断开连接）", client.ID),
								Timestamp: time.Now().Unix(),
							}
							if data, err := json.Marshal(notifyMsg); err == nil {
								for memberID := range group {
									m.trySendLocked(memberID, data)
								}
							}
						}
					}
				}

				client.Groups = make(map[string]bool)
				delete(m.Clients, client.ID)
				close(client.Send)
				m.mu.Unlock()
				logger.Debug(ctx, "ws client unregistered", "client_id", client.ID)
			}

		case message := <-m.Broadcast:
			switch message.Type {
			case MessageTypePrivate:
				m.handlePrivateMessage(message)
			case MessageTypeGroup:
				m.handleGroupMessage(message)
			case MessageTypeAnnouncement:
				m.handleAnnouncement(message)
			}
		}
	}
}

func (m *Manager) trySendLocked(clientID string, data []byte) {
	client, ok := m.Clients[clientID]
	if !ok {
		return
	}
	select {
	case client.Send <- data:
	default:
		logger.Warn(context.Background(), "ws send channel full", "client_id", clientID)
	}
}

func (m *Manager) handlePrivateMessage(message *Message) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.Marshal(message)
	if err != nil {
		logger.Error(context.Background(), "ws marshal private message failed", "error", err)
		return
	}

	m.trySendLocked(message.To, data)
	if message.From != message.To {
		m.trySendLocked(message.From, data)
	}
}

func (m *Manager) handleGroupMessage(message *Message) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	group, ok := m.Groups[message.To]
	if !ok {
		logger.Debug(context.Background(), "ws group not found", "group_id", message.To)
		return
	}

	data, err := json.Marshal(message)
	if err != nil {
		logger.Error(context.Background(), "ws marshal group message failed", "error", err)
		return
	}

	for userID := range group {
		m.trySendLocked(userID, data)
	}
}

func (m *Manager) handleAnnouncement(message *Message) {
	data, err := json.Marshal(message)
	if err != nil {
		logger.Error(context.Background(), "ws marshal announcement failed", "error", err)
		return
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	for id := range m.Clients {
		m.trySendLocked(id, data)
	}
}

// JoinGroup adds a client to a group
func (m *Manager) JoinGroup(groupID, clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.Groups[groupID]; !ok {
		m.Groups[groupID] = make(map[string]bool)
	}
	m.Groups[groupID][clientID] = true

	confirmMsg := &Message{
		Type:      MessageTypeAnnouncement,
		From:      clientID,
		Content:   fmt.Sprintf("用户 %s 加入了群组", clientID),
		Timestamp: time.Now().Unix(),
	}

	data, err := json.Marshal(confirmMsg)
	if err != nil {
		logger.Error(context.Background(), "ws marshal join confirmation failed", "error", err)
		return
	}

	for memberID := range m.Groups[groupID] {
		m.trySendLocked(memberID, data)
	}
}

// LeaveGroup removes a client from a group
func (m *Manager) LeaveGroup(groupID, clientID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	group, ok := m.Groups[groupID]
	if !ok {
		return
	}
	if !group[clientID] {
		return
	}

	delete(group, clientID)
	if len(group) == 0 {
		delete(m.Groups, groupID)
	}
	if client, ok := m.Clients[clientID]; ok {
		delete(client.Groups, groupID)
	}

	notifyMsg := &Message{
		Type:      MessageTypeAnnouncement,
		From:      clientID,
		Content:   fmt.Sprintf("用户 %s 退出了群组", clientID),
		Timestamp: time.Now().Unix(),
	}
	data, err := json.Marshal(notifyMsg)
	if err != nil {
		logger.Error(context.Background(), "ws marshal leave notification failed", "error", err)
		return
	}

	for memberID := range group {
		m.trySendLocked(memberID, data)
	}
	m.trySendLocked(clientID, data)
}

// WritePump handles writing messages to the WebSocket connection.
func (c *Client) WritePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		_ = c.Conn.Close(websocket.StatusNormalClosure, "")
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				_ = c.Conn.Close(websocket.StatusNormalClosure, "")
				return
			}

			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := c.Conn.Write(ctx, websocket.MessageText, message)
			cancel()
			if err != nil {
				logger.Warn(context.Background(), "ws write failed", "client_id", c.ID, "error", err)
				return
			}

		case <-ticker.C:
			ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
			err := c.Conn.Ping(ctx)
			cancel()
			if err != nil {
				logger.Debug(context.Background(), "ws ping failed", "client_id", c.ID, "error", err)
				return
			}
		}
	}
}

// ReadPump handles reading messages from the WebSocket connection.
func (c *Client) ReadPump() {
	defer func() {
		c.Manager.Unregister <- c
		_ = c.Conn.Close(websocket.StatusNormalClosure, "")
	}()

	c.Conn.SetReadLimit(512)

	for {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		_, message, err := c.Conn.Read(ctx)
		cancel()
		if err != nil {
			status := websocket.CloseStatus(err)
			if status == -1 || (status != websocket.StatusNormalClosure && status != websocket.StatusGoingAway) {
				logger.Warn(context.Background(), "ws read failed", "client_id", c.ID, "error", err)
			}
			break
		}

		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			logger.Warn(context.Background(), "ws invalid message json", "client_id", c.ID, "error", err)
			continue
		}

		if msg.From == "" {
			msg.From = c.ID
		}
		if msg.Timestamp == 0 {
			msg.Timestamp = time.Now().Unix()
		}

		if msg.Type < MessageTypePrivate || msg.Type > MessageTypeAnnouncement {
			logger.Warn(context.Background(), "ws invalid message type", "client_id", c.ID, "type", msg.Type)
			continue
		}

		c.Manager.Broadcast <- &msg
	}
}

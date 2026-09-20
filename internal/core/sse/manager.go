package sse

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go-admin-scaffold/pkg/logger"
)

// EventType defines the type of SSE event
const (
	EventTypeNotification = "notification"
	EventTypeAlert        = "alert"
	EventTypeUpdate       = "update"
)

// Event represents a server-sent event
type Event struct {
	ID      string      `json:"id"`
	Type    string      `json:"type"`
	Data    interface{} `json:"data"`
	Time    time.Time   `json:"time"`
	UserID  string      `json:"user_id,omitempty"`  // target user; empty = broadcast
	GroupID string      `json:"group_id,omitempty"` // target group; empty = not group-scoped
}

// Client represents an SSE client connection
type Client struct {
	ID       string
	Groups   map[string]bool
	Messages chan *Event
}

// Manager manages SSE connections and event broadcasting
type Manager struct {
	clients    map[string]*Client
	groups     map[string]map[string]bool // group -> userIDs
	register   chan *Client
	unregister chan *Client
	events     chan *Event
	mu         sync.RWMutex
}

// NewManager creates a new SSE manager
func NewManager() *Manager {
	return &Manager{
		clients:    make(map[string]*Client),
		groups:     make(map[string]map[string]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		events:     make(chan *Event, 100),
	}
}

// Start starts the SSE manager
func (m *Manager) Start() {
	ctx := context.Background()
	for {
		select {
		case client := <-m.register:
			m.mu.Lock()
			m.clients[client.ID] = client
			m.mu.Unlock()
			logger.Debug(ctx, "sse client registered", "client_id", client.ID)

		case client := <-m.unregister:
			m.mu.Lock()
			if _, ok := m.clients[client.ID]; ok {
				for groupID := range client.Groups {
					if group, exists := m.groups[groupID]; exists {
						delete(group, client.ID)
						if len(group) == 0 {
							delete(m.groups, groupID)
						}
					}
				}
				delete(m.clients, client.ID)
				close(client.Messages)
				logger.Debug(ctx, "sse client unregistered", "client_id", client.ID)
			}
			m.mu.Unlock()

		case event := <-m.events:
			m.handleEvent(event)
		}
	}
}

func (m *Manager) trySend(client *Client, event *Event) {
	select {
	case client.Messages <- event:
	default:
		logger.Warn(context.Background(), "sse send channel full", "client_id", client.ID)
	}
}

// handleEvent processes and distributes events to relevant clients
func (m *Manager) handleEvent(event *Event) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if event.UserID != "" {
		if client, ok := m.clients[event.UserID]; ok {
			m.trySend(client, event)
		}
		return
	}

	if event.GroupID != "" {
		if group, ok := m.groups[event.GroupID]; ok {
			for userID := range group {
				if client, ok := m.clients[userID]; ok {
					m.trySend(client, event)
				}
			}
		}
		return
	}

	for _, client := range m.clients {
		m.trySend(client, event)
	}
}

// Register registers a new client
func (m *Manager) Register(userID string) *Client {
	client := &Client{
		ID:       userID,
		Groups:   make(map[string]bool),
		Messages: make(chan *Event, 100),
	}
	m.register <- client
	return client
}

// Unregister removes a client
func (m *Manager) Unregister(client *Client) {
	m.unregister <- client
}

// JoinGroup adds a client to a group
func (m *Manager) JoinGroup(groupID, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, ok := m.groups[groupID]; !ok {
		m.groups[groupID] = make(map[string]bool)
	}
	m.groups[groupID][userID] = true

	if client, ok := m.clients[userID]; ok {
		client.Groups[groupID] = true
	}

	m.events <- &Event{
		ID:      fmt.Sprintf("join_%s_%d", groupID, time.Now().UnixNano()),
		Type:    EventTypeNotification,
		Data:    fmt.Sprintf("用户 %s 加入了组 %s", userID, groupID),
		Time:    time.Now(),
		GroupID: groupID,
	}
}

// LeaveGroup removes a client from a group
func (m *Manager) LeaveGroup(groupID, userID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if group, ok := m.groups[groupID]; ok {
		delete(group, userID)
		if len(group) == 0 {
			delete(m.groups, groupID)
		}
	}

	if client, ok := m.clients[userID]; ok {
		delete(client.Groups, groupID)
	}

	m.events <- &Event{
		ID:      fmt.Sprintf("leave_%s_%d", groupID, time.Now().UnixNano()),
		Type:    EventTypeNotification,
		Data:    fmt.Sprintf("用户 %s 离开了组 %s", userID, groupID),
		Time:    time.Now(),
		GroupID: groupID,
	}
}

// SendEvent sends an event to the specified target(s)
func (m *Manager) SendEvent(event *Event) {
	if event.ID == "" {
		event.ID = fmt.Sprintf("evt_%d", time.Now().UnixNano())
	}
	if event.Time.IsZero() {
		event.Time = time.Now()
	}
	m.events <- event
}

// Broadcast sends an event to all connected clients
func (m *Manager) Broadcast(eventType string, data interface{}) {
	event := &Event{
		ID:   fmt.Sprintf("broadcast_%d", time.Now().UnixNano()),
		Type: eventType,
		Data: data,
		Time: time.Now(),
	}
	m.events <- event
}

// SendToUser sends an event to a specific user
func (m *Manager) SendToUser(userID, eventType string, data interface{}) {
	event := &Event{
		ID:     fmt.Sprintf("user_%s_%d", userID, time.Now().UnixNano()),
		Type:   eventType,
		Data:   data,
		Time:   time.Now(),
		UserID: userID,
	}
	m.events <- event
}

// SendToGroup sends an event to all members of a group
func (m *Manager) SendToGroup(groupID, eventType string, data interface{}) {
	event := &Event{
		ID:      fmt.Sprintf("group_%s_%d", groupID, time.Now().UnixNano()),
		Type:    eventType,
		Data:    data,
		Time:    time.Now(),
		GroupID: groupID,
	}
	m.events <- event
}

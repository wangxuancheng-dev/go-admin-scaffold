package ws

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestManager_JoinLeaveGroup(t *testing.T) {
	m := NewManager()
	go m.Start()

	send := make(chan []byte, 8)
	client := &Client{
		ID:      "u1",
		Send:    send,
		Manager: m,
		Groups:  make(map[string]bool),
	}
	m.Register <- client

	// wait until registered
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		m.mu.RLock()
		_, ok := m.Clients["u1"]
		m.mu.RUnlock()
		if ok {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	m.JoinGroup("g1", "u1")
	client.Groups["g1"] = true

	m.mu.RLock()
	_, inGroup := m.Groups["g1"]["u1"]
	m.mu.RUnlock()
	require.True(t, inGroup)

	// drain join announcement
	select {
	case <-send:
	case <-time.After(200 * time.Millisecond):
	}

	m.LeaveGroup("g1", "u1")
	m.mu.RLock()
	_, groupExists := m.Groups["g1"]
	m.mu.RUnlock()
	require.False(t, groupExists)
}

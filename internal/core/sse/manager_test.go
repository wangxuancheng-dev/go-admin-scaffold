package sse

import (
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestManager_RegisterSendToUser(t *testing.T) {
	m := NewManager()
	go m.Start()

	client := m.Register("u1")
	deadline := time.Now().Add(time.Second)
	for time.Now().Before(deadline) {
		m.mu.RLock()
		_, ok := m.clients["u1"]
		m.mu.RUnlock()
		if ok {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	m.SendToUser("u1", EventTypeNotification, "hello")

	select {
	case ev := <-client.Messages:
		require.Equal(t, EventTypeNotification, ev.Type)
		require.Equal(t, "hello", ev.Data)
		require.Equal(t, "u1", ev.UserID)
	case <-time.After(time.Second):
		t.Fatal("expected event")
	}

	m.Unregister(client)
}

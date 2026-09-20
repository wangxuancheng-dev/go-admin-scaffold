package services

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRealtimeTicketService_IssueConsume(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	svc := NewRealtimeTicketService(rdb, 30*time.Second)

	ctx := context.Background()
	ticket, ttl, err := svc.Issue(ctx, 42)
	require.NoError(t, err)
	require.NotEmpty(t, ticket)
	assert.Equal(t, 30*time.Second, ttl)

	uid, err := svc.Consume(ctx, ticket)
	require.NoError(t, err)
	assert.Equal(t, uint(42), uid)

	_, err = svc.Consume(ctx, ticket)
	assert.Error(t, err)

	_, err = svc.Consume(ctx, "")
	assert.Error(t, err)
}

func TestRealtimeTicketService_unavailable(t *testing.T) {
	var svc *RealtimeTicketService
	_, _, err := svc.Issue(context.Background(), 1)
	assert.Error(t, err)

	svc = NewRealtimeTicketService(nil, 0)
	_, _, err = svc.Issue(context.Background(), 1)
	assert.Error(t, err)
}

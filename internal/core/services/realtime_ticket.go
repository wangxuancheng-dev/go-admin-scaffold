package services

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const realtimeTicketKeyPrefix = "realtime:ticket:"

// RealtimeTicketService issues one-time short-lived connect tickets (Redis).
// Prefer these over putting JWTs in WebSocket/SSE URLs.
type RealtimeTicketService struct {
	rdb *redis.Client
	ttl time.Duration
}

func NewRealtimeTicketService(rdb *redis.Client, ttl time.Duration) *RealtimeTicketService {
	if ttl <= 0 {
		ttl = 60 * time.Second
	}
	return &RealtimeTicketService{rdb: rdb, ttl: ttl}
}

// Issue stores userID under a random ticket id and returns the ticket.
func (s *RealtimeTicketService) Issue(ctx context.Context, userID uint) (string, time.Duration, error) {
	if s == nil || s.rdb == nil {
		return "", 0, fmt.Errorf("realtime ticket store unavailable")
	}
	ticket := uuid.NewString()
	key := realtimeTicketKeyPrefix + ticket
	if err := s.rdb.Set(ctx, key, strconv.FormatUint(uint64(userID), 10), s.ttl).Err(); err != nil {
		return "", 0, err
	}
	return ticket, s.ttl, nil
}

// Consume deletes the ticket and returns the user id (one-time use).
func (s *RealtimeTicketService) Consume(ctx context.Context, ticket string) (uint, error) {
	if s == nil || s.rdb == nil {
		return 0, fmt.Errorf("realtime ticket store unavailable")
	}
	if ticket == "" {
		return 0, fmt.Errorf("empty ticket")
	}
	key := realtimeTicketKeyPrefix + ticket
	uidStr, err := s.rdb.GetDel(ctx, key).Result()
	if err == redis.Nil {
		return 0, fmt.Errorf("ticket not found or expired")
	}
	if err != nil {
		return 0, err
	}
	id, err := strconv.ParseUint(uidStr, 10, 32)
	if err != nil {
		return 0, fmt.Errorf("invalid ticket payload")
	}
	return uint(id), nil
}

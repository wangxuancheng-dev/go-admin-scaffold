package services

import (
	"context"
	"testing"
	"time"

	"go-admin-scaffold/internal/core/models"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

func TestRealtimeTicketService_issueConsume(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	t.Cleanup(mr.Close)

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	t.Cleanup(func() { _ = rdb.Close() })

	svc := NewRealtimeTicketService(rdb, time.Second)
	ticket, ttl, err := svc.Issue(context.Background(), 42)
	require.NoError(t, err)
	require.NotEmpty(t, ticket)
	require.Equal(t, time.Second, ttl)

	uid, err := svc.Consume(context.Background(), ticket)
	require.NoError(t, err)
	require.Equal(t, uint(42), uid)

	_, err = svc.Consume(context.Background(), ticket)
	require.Error(t, err)
}

func TestAuthService_LoginAndRefresh(t *testing.T) {
	hasher := NewAuthService(&stubUserRepo{}, nil, testAuthConfig("0123456789abcdef0123456789abcdef", nil))
	hash, err := hasher.HashPassword("secret")
	require.NoError(t, err)

	repo := &stubUserRepo{user: &models.User{ID: 7, Username: "alice", Password: hash, Status: 1}}
	svc := NewAuthService(repo, nil, testAuthConfig("0123456789abcdef0123456789abcdef", nil))

	tok, err := svc.Login(context.Background(), &LoginRequest{
		Username: "alice", Password: "secret", CaptchaID: "c", CaptchaCode: "1",
	})
	require.NoError(t, err)
	require.NotEmpty(t, tok.AccessToken)

	_, err = svc.Login(context.Background(), &LoginRequest{
		Username: "alice", Password: "wrong", CaptchaID: "c", CaptchaCode: "1",
	})
	require.ErrorIs(t, err, ErrInvalidCredentials)

	refreshed, err := svc.RefreshToken(context.Background(), 7)
	require.NoError(t, err)
	require.NotEmpty(t, refreshed)

	user, err := svc.GetUserByID(context.Background(), 7)
	require.NoError(t, err)
	require.Equal(t, "alice", user.Username)

	require.NoError(t, svc.Logout(context.Background(), 7))
}

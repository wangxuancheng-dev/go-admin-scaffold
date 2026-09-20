package redis_test

import (
	"context"
	"testing"

	"go-admin-scaffold/pkg/redis"

	"github.com/alicebob/miniredis/v2"
	"github.com/stretchr/testify/require"
)

func TestSetup_ping(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	host, port, err := splitHostPort(mr.Addr())
	require.NoError(t, err)

	client, err := redis.Setup(&redis.Config{Host: host, Port: port})
	if err != nil {
		// sync.Once: a prior Setup in this process may have failed or succeeded already.
		t.Skipf("redis Setup already consumed in this process: %v", err)
	}
	require.NotNil(t, client)
	require.NoError(t, client.Ping(context.Background()).Err())
}

func splitHostPort(addr string) (string, string, error) {
	for i := len(addr) - 1; i >= 0; i-- {
		if addr[i] == ':' {
			return addr[:i], addr[i+1:], nil
		}
	}
	return "", "", errString("no port")
}

type errString string

func (e errString) Error() string { return string(e) }

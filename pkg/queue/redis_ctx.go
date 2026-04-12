package queue

import "context"

type redisCtxKey string

const redisConsumerKey redisCtxKey = "redis_stream_consumer"

// WithRedisConsumer attaches a unique consumer name for Redis Streams (XREADGROUP).
func WithRedisConsumer(ctx context.Context, consumer string) context.Context {
	return context.WithValue(ctx, redisConsumerKey, consumer)
}

func redisConsumerFromContext(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if v := ctx.Value(redisConsumerKey); v != nil {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

package queue

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
)

// Lua: atomically move due delayed ZSET members into the stream.
const redisPromoteDelayedScript = `
local delayed = KEYS[1]
local stream = KEYS[2]
local now = tonumber(ARGV[1])
local limit = tonumber(ARGV[2])
local members = redis.call('ZRANGEBYSCORE', delayed, '0', now, 'LIMIT', 0, limit)
local n = 0
for _, m in ipairs(members) do
  redis.call('ZREM', delayed, m)
  redis.call('XADD', stream, '*', 'data', m)
  n = n + 1
end
return n
`

var (
	redisPromoteScript = redis.NewScript(redisPromoteDelayedScript)
	redisDefaultGroup  = "queue_workers"
)

// RedisQueue implements QueueInterface using Redis Streams (XADD / XREADGROUP / XACK) plus a ZSET for delayed jobs.
type RedisQueue struct {
	client          *redis.Client
	config          Config
	group           string
	defaultConsumer string
}

// NewRedisQueue creates a Redis queue backed by Streams + consumer group.
func NewRedisQueue(config Config) (*RedisQueue, error) {
	connection, ok := config.Options["connection"].(string)
	if !ok {
		return nil, fmt.Errorf("redis connection string not found in options")
	}

	opt, err := redis.ParseURL(connection)
	if err != nil {
		return nil, fmt.Errorf("invalid redis connection string: %v", err)
	}

	client := redis.NewClient(opt)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to redis: %v", err)
	}

	group := redisDefaultGroup
	if g, ok := config.Options["stream_group"].(string); ok && g != "" {
		group = g
	}

	hostname, _ := os.Hostname()
	if hostname == "" {
		hostname = "unknown"
	}
	defaultConsumer := fmt.Sprintf("%s-%d", hostname, os.Getpid())
	if c, ok := config.Options["consumer"].(string); ok && c != "" {
		defaultConsumer = c
	}

	return &RedisQueue{
		client:          client,
		config:          config,
		group:           group,
		defaultConsumer: defaultConsumer,
	}, nil
}

func (q *RedisQueue) resolveQueue(queue string) (string, error) {
	if queue != "" {
		return queue, nil
	}
	if queueOpt, ok := q.config.Options["queue"].(string); ok {
		return queueOpt, nil
	}
	return "", fmt.Errorf("queue name not found in options")
}

func (q *RedisQueue) streamKey(queue string) string {
	return fmt.Sprintf("queues:%s:stream", queue)
}

func (q *RedisQueue) delayedKey(queue string) string {
	return fmt.Sprintf("queues:%s:delayed", queue)
}

func (q *RedisQueue) ensureGroup(ctx context.Context, stream string) error {
	err := q.client.XGroupCreateMkStream(ctx, stream, q.group, "0").Err()
	if err == nil {
		return nil
	}
	msg := strings.ToLower(err.Error())
	if strings.Contains(msg, "busygroup") {
		return nil
	}
	return err
}

func (q *RedisQueue) consumerForPop(ctx context.Context) string {
	if c := redisConsumerFromContext(ctx); c != "" {
		return c
	}
	return q.defaultConsumer
}

func isRedisStreamMessageID(id string) bool {
	if id == "" {
		return false
	}
	parts := strings.Split(id, "-")
	if len(parts) != 2 {
		return false
	}
	_, err1 := strconv.ParseUint(parts[0], 10, 64)
	_, err2 := strconv.ParseUint(parts[1], 10, 64)
	return err1 == nil && err2 == nil
}

func (q *RedisQueue) promoteDelayed(ctx context.Context, queue string) error {
	dk := q.delayedKey(queue)
	sk := q.streamKey(queue)
	now := time.Now().Unix()
	_, err := redisPromoteScript.Run(ctx, q.client, []string{dk, sk}, now, 100).Result()
	return err
}

func redisUniqueLockKey(queue, uniqueKey string) string {
	sum := sha256.Sum256([]byte(uniqueKey))
	return fmt.Sprintf("queues:%s:unique:%x", queue, sum[:12])
}

func mergeUniqueKeyIntoPayload(payload []byte, uniqueKey string) []byte {
	uk := NormalizeUniqueKey(uniqueKey)
	if uk == "" {
		return payload
	}
	var m map[string]interface{}
	if err := json.Unmarshal(payload, &m); err != nil {
		return payload
	}
	if _, exists := m["unique_key"]; !exists {
		m["unique_key"] = uk
		out, err := json.Marshal(m)
		if err != nil {
			return payload
		}
		return out
	}
	return payload
}

func (q *RedisQueue) uniqueLockTTL(job JobInterface) time.Duration {
	const def = 24 * time.Hour
	var fromConfig time.Duration
	if v, ok := q.config.Options["unique_ttl"]; ok {
		switch t := v.(type) {
		case int:
			if t > 0 {
				fromConfig = time.Duration(t) * time.Second
			}
		case int64:
			if t > 0 {
				fromConfig = time.Duration(t) * time.Second
			}
		case float64:
			if t > 0 {
				fromConfig = time.Duration(t) * time.Second
			}
		case time.Duration:
			if t > 0 {
				fromConfig = t
			}
		}
	}
	if fromConfig > 0 {
		return fromConfig
	}
	if job != nil {
		to := job.GetTimeout()
		if to > 0 && to*3 > def {
			return to * 3
		}
	}
	return def
}

func (q *RedisQueue) acquireUniqueString(ctx context.Context, queue, uniqueKey string, ttl time.Duration) error {
	uk := NormalizeUniqueKey(uniqueKey)
	if uk == "" {
		return nil
	}
	ok, err := q.client.SetNX(ctx, redisUniqueLockKey(queue, uk), "1", ttl).Result()
	if err != nil {
		return err
	}
	if !ok {
		return ErrDuplicateJob
	}
	return nil
}

func (q *RedisQueue) acquireUnique(ctx context.Context, queue string, job JobInterface) error {
	return q.acquireUniqueString(ctx, queue, job.GetUniqueKey(), q.uniqueLockTTL(job))
}

func (q *RedisQueue) releaseUniqueString(ctx context.Context, queue, uniqueKey string) error {
	uk := NormalizeUniqueKey(uniqueKey)
	if uk == "" {
		return nil
	}
	return q.client.Del(ctx, redisUniqueLockKey(queue, uk)).Err()
}

func (q *RedisQueue) releaseUnique(ctx context.Context, queue string, job JobInterface) error {
	return q.releaseUniqueString(ctx, queue, job.GetUniqueKey())
}

func (q *RedisQueue) enqueueDelayedMember(ctx context.Context, queue, data string, runAt time.Time) error {
	score := float64(runAt.Unix())
	return q.client.ZAdd(ctx, q.delayedKey(queue), redis.Z{
		Score:  score,
		Member: data,
	}).Err()
}

func (q *RedisQueue) enqueueStreamData(ctx context.Context, queue, data string) error {
	_, err := q.client.XAdd(ctx, &redis.XAddArgs{
		Stream: q.streamKey(queue),
		Values: map[string]interface{}{"data": data},
	}).Result()
	return err
}

// Push appends a job to the stream (or delayed ZSET if delayed).
func (q *RedisQueue) Push(ctx context.Context, job JobInterface) error {
	queue, err := q.resolveQueue(job.GetQueue())
	if err != nil {
		return err
	}

	if err := q.acquireUnique(ctx, queue, job); err != nil {
		return err
	}

	payload, err := json.Marshal(job)
	if err != nil {
		_ = q.releaseUnique(ctx, queue, job)
		return err
	}
	data := string(payload)

	if job.GetDelay() > 0 {
		if err := q.enqueueDelayedMember(ctx, queue, data, time.Now().Add(job.GetDelay())); err != nil {
			_ = q.releaseUnique(ctx, queue, job)
			return err
		}
		return nil
	}

	if err := q.enqueueStreamData(ctx, queue, data); err != nil {
		_ = q.releaseUnique(ctx, queue, job)
		return err
	}
	return nil
}

// PushRaw pushes raw JSON payload to the stream or delayed set.
func (q *RedisQueue) PushRaw(ctx context.Context, queue string, payload []byte, options map[string]interface{}) error {
	queue, err := q.resolveQueue(queue)
	if err != nil {
		return err
	}

	delay := time.Duration(0)
	if v, ok := options["delay"].(time.Duration); ok {
		delay = v
	}

	ukFromOpt := ""
	if v, ok := options["unique_key"].(string); ok {
		ukFromOpt = v
	}
	payload = mergeUniqueKeyIntoPayload(payload, ukFromOpt)

	if err := q.acquireUniqueString(ctx, queue, ukFromOpt, q.uniqueLockTTL(nil)); err != nil {
		return err
	}

	if delay > 0 {
		if err := q.enqueueDelayedMember(ctx, queue, string(payload), time.Now().Add(delay)); err != nil {
			_ = q.releaseUniqueString(ctx, queue, ukFromOpt)
			return err
		}
		return nil
	}

	if err := q.enqueueStreamData(ctx, queue, string(payload)); err != nil {
		_ = q.releaseUniqueString(ctx, queue, ukFromOpt)
		return err
	}
	return nil
}

// Later schedules a job in the delayed ZSET.
func (q *RedisQueue) Later(ctx context.Context, job JobInterface, delay time.Duration) error {
	queue, err := q.resolveQueue(job.GetQueue())
	if err != nil {
		return err
	}

	if err := q.acquireUnique(ctx, queue, job); err != nil {
		return err
	}

	payload, err := json.Marshal(job)
	if err != nil {
		_ = q.releaseUnique(ctx, queue, job)
		return err
	}

	if err := q.enqueueDelayedMember(ctx, queue, string(payload), time.Now().Add(delay)); err != nil {
		_ = q.releaseUnique(ctx, queue, job)
		return err
	}
	return nil
}

// Pop reads one message with XREADGROUP (at-least-once). Delayed jobs are promoted into the stream first.
func (q *RedisQueue) Pop(ctx context.Context, queue string) (JobInterface, error) {
	queue, err := q.resolveQueue(queue)
	if err != nil {
		return nil, err
	}

	sk := q.streamKey(queue)
	if err := q.ensureGroup(ctx, sk); err != nil {
		return nil, err
	}

	if err := q.promoteDelayed(ctx, queue); err != nil {
		return nil, err
	}

	consumer := q.consumerForPop(ctx)

	// Deliver pending entries for this consumer first (crash recovery with stable consumer name).
	pending, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.group,
		Consumer: consumer,
		Streams:  []string{sk, "0"},
		Count:    10,
		Block:    0,
		NoAck:    false,
	}).Result()
	if err != nil && err != redis.Nil {
		return nil, err
	}
	if len(pending) > 0 && len(pending[0].Messages) > 0 {
		return q.parseStreamMessage(ctx, sk, pending[0].Messages[0])
	}

	res, err := q.client.XReadGroup(ctx, &redis.XReadGroupArgs{
		Group:    q.group,
		Consumer: consumer,
		Streams:  []string{sk, ">"},
		Count:    1,
		Block:    800 * time.Millisecond,
		NoAck:    false,
	}).Result()

	if err != nil {
		if err == redis.Nil {
			return nil, ErrQueueEmpty
		}
		return nil, err
	}

	if len(res) == 0 || len(res[0].Messages) == 0 {
		return nil, ErrQueueEmpty
	}

	return q.parseStreamMessage(ctx, sk, res[0].Messages[0])
}

func (q *RedisQueue) parseStreamMessage(ctx context.Context, stream string, msg redis.XMessage) (JobInterface, error) {
	raw, ok := msg.Values["data"].(string)
	if !ok {
		_ = q.client.XAck(ctx, stream, q.group, msg.ID)
		_ = q.client.XDel(ctx, stream, msg.ID)
		return nil, fmt.Errorf("redis stream message missing data field")
	}

	var job BaseJob
	if err := json.Unmarshal([]byte(raw), &job); err != nil {
		_ = q.client.XAck(ctx, stream, q.group, msg.ID)
		_ = q.client.XDel(ctx, stream, msg.ID)
		return nil, err
	}

	job.SetID(msg.ID)
	return &job, nil
}

// Size returns stream length + delayed ZSET cardinality (does not include unacked pending).
func (q *RedisQueue) Size(ctx context.Context, queue string) (int64, error) {
	queue, err := q.resolveQueue(queue)
	if err != nil {
		return 0, err
	}

	sk := q.streamKey(queue)
	streamLen, err := q.client.XLen(ctx, sk).Result()
	if err != nil {
		return 0, err
	}

	delayed, err := q.client.ZCard(ctx, q.delayedKey(queue)).Result()
	if err != nil {
		return 0, err
	}

	return streamLen + delayed, nil
}

// Delete acknowledges and removes a stream message, or removes a delayed member by payload match.
func (q *RedisQueue) Delete(ctx context.Context, queue string, job JobInterface) error {
	queue, err := q.resolveQueue(queue)
	if err != nil {
		return err
	}

	sk := q.streamKey(queue)
	id := job.GetID()
	if isRedisStreamMessageID(id) {
		if err := q.client.XAck(ctx, sk, q.group, id).Err(); err != nil {
			return err
		}
		if err := q.client.XDel(ctx, sk, id).Err(); err != nil {
			return err
		}
		return q.releaseUnique(ctx, queue, job)
	}

	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	if err := q.client.ZRem(ctx, q.delayedKey(queue), string(payload)).Err(); err != nil {
		return err
	}
	return q.releaseUnique(ctx, queue, job)
}

// Release ACKs the stream message (if any) and re-schedules the job (delayed ZSET or immediate XADD).
func (q *RedisQueue) Release(ctx context.Context, queue string, job JobInterface, delay time.Duration) error {
	queue, err := q.resolveQueue(queue)
	if err != nil {
		return err
	}

	sk := q.streamKey(queue)
	id := job.GetID()
	if isRedisStreamMessageID(id) {
		if err := q.client.XAck(ctx, sk, q.group, id).Err(); err != nil {
			return err
		}
		_ = q.client.XDel(ctx, sk, id)
	}

	job.SetAttempts(job.GetAttempts() + 1)
	if job.GetAttempts() >= job.GetMaxAttempts() {
		return q.releaseUnique(ctx, queue, job)
	}

	newDelay := delay
	if newDelay == 0 {
		backoff := job.GetBackoff()
		if len(backoff) > 0 {
			attempt := job.GetAttempts() - 1
			if attempt < len(backoff) {
				newDelay = backoff[attempt]
			} else {
				newDelay = backoff[len(backoff)-1]
			}
		} else {
			newDelay = job.GetRetryAfter()
		}
	}

	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}
	data := string(payload)

	if newDelay > 0 {
		return q.enqueueDelayedMember(ctx, queue, data, time.Now().Add(newDelay))
	}

	return q.enqueueStreamData(ctx, queue, data)
}

// Clear deletes stream, delayed ZSET, and consumer group state for the queue.
func (q *RedisQueue) Clear(ctx context.Context, queue string) error {
	queue, err := q.resolveQueue(queue)
	if err != nil {
		return err
	}

	sk := q.streamKey(queue)
	_ = q.client.XGroupDestroy(ctx, sk, q.group).Err()

	prefix := fmt.Sprintf("queues:%s:unique:", queue)
	var cursor uint64
	for {
		keys, next, err := q.client.Scan(ctx, cursor, prefix+"*", 256).Result()
		if err != nil {
			return err
		}
		if len(keys) > 0 {
			if err := q.client.Del(ctx, keys...).Err(); err != nil {
				return err
			}
		}
		cursor = next
		if cursor == 0 {
			break
		}
	}

	pipe := q.client.Pipeline()
	pipe.Del(ctx, sk)
	pipe.Del(ctx, q.delayedKey(queue))
	_, err = pipe.Exec(ctx)
	return err
}

// Close closes the Redis client.
func (q *RedisQueue) Close() error {
	return q.client.Close()
}

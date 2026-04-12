package queue

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/hibiken/asynq"
	"github.com/redis/go-redis/v9"
)

const defaultAsynqTaskType = "job"

// isAsynqQueueNotExistError matches asynq internal *errors.QueueNotFoundError (not exported).
// Before any task is enqueued, Asynq has no queue record and Inspector ops fail accordingly.
func isAsynqQueueNotExistError(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, `does not exist`) && strings.Contains(s, "queue")
}

// AsynqQueue implements QueueInterface using github.com/hibiken/asynq (Redis).
type AsynqQueue struct {
	client       *asynq.Client
	inspector    *asynq.Inspector
	config       Config
	defaultQueue string
}

// NewAsynqQueue builds an Asynq-backed queue from Config (driver redis / asynq).
func NewAsynqQueue(config Config) (*AsynqQueue, error) {
	connection, ok := config.Options["connection"].(string)
	if !ok || connection == "" {
		return nil, fmt.Errorf("redis connection string not found in options")
	}

	parsed, err := redis.ParseURL(connection)
	if err != nil {
		return nil, fmt.Errorf("invalid redis connection string: %w", err)
	}

	rdb := redis.NewClient(parsed)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := rdb.Ping(ctx).Err(); err != nil {
		_ = rdb.Close()
		return nil, fmt.Errorf("failed to connect to redis (asynq): %w", err)
	}
	_ = rdb.Close()

	redisOpt := asynq.RedisClientOpt{
		Addr:      parsed.Addr,
		Username:  parsed.Username,
		Password:  parsed.Password,
		DB:        parsed.DB,
		TLSConfig: parsed.TLSConfig,
	}

	client := asynq.NewClient(redisOpt)
	inspector := asynq.NewInspector(redisOpt)

	dq := "default"
	if q, ok := config.Options["queue"].(string); ok && q != "" {
		dq = q
	}

	return &AsynqQueue{
		client:       client,
		inspector:    inspector,
		config:       config,
		defaultQueue: dq,
	}, nil
}

func (q *AsynqQueue) resolveQueue(name string) (string, error) {
	if name != "" {
		return name, nil
	}
	if q.defaultQueue != "" {
		return q.defaultQueue, nil
	}
	return "", fmt.Errorf("queue name not found in options")
}

func taskTypeForJob(job JobInterface) (string, error) {
	ensureJobTypeForPush(job)
	if job == nil {
		return "", fmt.Errorf("nil job")
	}
	t := job.TaskType()
	if t == "" {
		t = defaultAsynqTaskType
	}
	return t, nil
}

func asynqMaxRetry(job JobInterface) int {
	n := job.GetMaxAttempts()
	if n < 1 {
		n = 1
	}
	return n - 1
}

// uniqueAsynqTaskID builds a stable Asynq task ID for business unique_key.
// Asynq's Unique() dedupes by queue + type + full JSON payload; timestamps/message differ → no match.
// TaskID collision matches Laravel-style "same key while task exists" without identical payloads.
func uniqueAsynqTaskID(queueName, taskType, uniqueKey string) string {
	h := sha256.Sum256([]byte("uk:v1\x00" + queueName + "\x00" + taskType + "\x00" + uniqueKey))
	return "uk:" + hex.EncodeToString(h[:])
}

func asynqMergeUniqueKeyIntoPayload(payload []byte, uniqueKey string) []byte {
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

// Push enqueues a task (full job JSON as payload; task type = job_type).
func (q *AsynqQueue) Push(ctx context.Context, job JobInterface) error {
	queueName, err := q.resolveQueue(job.GetQueue())
	if err != nil {
		return err
	}

	typ, err := taskTypeForJob(job)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}

	task := asynq.NewTask(typ, payload)
	opts := []asynq.Option{
		asynq.Queue(queueName),
		asynq.MaxRetry(asynqMaxRetry(job)),
	}
	if to := job.GetTimeout(); to > 0 {
		opts = append(opts, asynq.Timeout(to))
	}
	if d := job.GetDelay(); d > 0 {
		opts = append(opts, asynq.ProcessIn(d))
	}
	if uk := NormalizeUniqueKey(job.GetUniqueKey()); uk != "" {
		opts = append(opts, asynq.TaskID(uniqueAsynqTaskID(queueName, typ, uk)))
	}

	_, err = q.client.EnqueueContext(ctx, task, opts...)
	if errors.Is(err, asynq.ErrDuplicateTask) || errors.Is(err, asynq.ErrTaskIDConflict) {
		return ErrDuplicateJob
	}
	return err
}

// PushRaw enqueues a task with optional task_type (default "raw"), delay, retries, timeout, unique_key.
func (q *AsynqQueue) PushRaw(ctx context.Context, queue string, payload []byte, options map[string]interface{}) error {
	queueName, err := q.resolveQueue(queue)
	if err != nil {
		return err
	}

	typ := "raw"
	if options != nil {
		if v, ok := options["task_type"].(string); ok && v != "" {
			typ = v
		}
	}

	delay := time.Duration(0)
	maxAttempts := 3
	timeout := time.Duration(0)
	ukFromOpt := ""
	if options != nil {
		if v, ok := options["delay"].(time.Duration); ok {
			delay = v
		}
		if v, ok := options["max_attempts"].(int); ok && v > 0 {
			maxAttempts = v
		}
		if v, ok := options["timeout"].(time.Duration); ok {
			timeout = v
		}
		if v, ok := options["unique_key"].(string); ok {
			ukFromOpt = v
		}
	}
	payload = asynqMergeUniqueKeyIntoPayload(payload, ukFromOpt)

	task := asynq.NewTask(typ, payload)
	opts := []asynq.Option{
		asynq.Queue(queueName),
		asynq.MaxRetry(maxAttempts - 1),
	}
	if timeout > 0 {
		opts = append(opts, asynq.Timeout(timeout))
	}
	if delay > 0 {
		opts = append(opts, asynq.ProcessIn(delay))
	}
	if uk := NormalizeUniqueKey(ukFromOpt); uk != "" {
		opts = append(opts, asynq.TaskID(uniqueAsynqTaskID(queueName, typ, uk)))
	}

	_, err = q.client.EnqueueContext(ctx, task, opts...)
	if errors.Is(err, asynq.ErrDuplicateTask) || errors.Is(err, asynq.ErrTaskIDConflict) {
		return ErrDuplicateJob
	}
	return err
}

// Later schedules a job to run after delay.
func (q *AsynqQueue) Later(ctx context.Context, job JobInterface, delay time.Duration) error {
	queueName, err := q.resolveQueue(job.GetQueue())
	if err != nil {
		return err
	}

	typ, err := taskTypeForJob(job)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}

	task := asynq.NewTask(typ, payload)
	opts := []asynq.Option{
		asynq.Queue(queueName),
		asynq.MaxRetry(asynqMaxRetry(job)),
		asynq.ProcessIn(delay),
	}
	if to := job.GetTimeout(); to > 0 {
		opts = append(opts, asynq.Timeout(to))
	}
	if uk := NormalizeUniqueKey(job.GetUniqueKey()); uk != "" {
		opts = append(opts, asynq.TaskID(uniqueAsynqTaskID(queueName, typ, uk)))
	}

	_, err = q.client.EnqueueContext(ctx, task, opts...)
	if errors.Is(err, asynq.ErrDuplicateTask) || errors.Is(err, asynq.ErrTaskIDConflict) {
		return ErrDuplicateJob
	}
	return err
}

// Pop is not supported with Asynq; use asynq.Server to consume tasks.
func (q *AsynqQueue) Pop(ctx context.Context, queue string) (JobInterface, error) {
	_ = ctx
	_ = queue
	return nil, fmt.Errorf("%w", ErrPullNotSupported)
}

// Size returns total tasks in the queue (pending + active + scheduled + retry + archived + aggregating + completed per asynq.QueueInfo).
func (q *AsynqQueue) Size(ctx context.Context, queue string) (int64, error) {
	queueName, err := q.resolveQueue(queue)
	if err != nil {
		return 0, err
	}
	info, err := q.inspector.GetQueueInfo(queueName)
	if err != nil {
		if isAsynqQueueNotExistError(err) {
			return 0, nil
		}
		return 0, err
	}
	return int64(info.Size), nil
}

// Delete is not supported without the Asynq task id; use asynq.Inspector in application code if needed.
func (q *AsynqQueue) Delete(ctx context.Context, queue string, job JobInterface) error {
	_ = ctx
	_ = queue
	_ = job
	return fmt.Errorf("%w", ErrPullNotSupported)
}

// Release is not supported; Asynq manages retries after handler errors.
func (q *AsynqQueue) Release(ctx context.Context, queue string, job JobInterface, delay time.Duration) error {
	_ = ctx
	_ = queue
	_ = job
	_ = delay
	return fmt.Errorf("%w", ErrPullNotSupported)
}

// Clear removes pending, scheduled, retry, archived, and completed tasks from the queue name.
func (q *AsynqQueue) Clear(ctx context.Context, queue string) error {
	queueName, err := q.resolveQueue(queue)
	if err != nil {
		return err
	}

	if _, err := q.inspector.GetQueueInfo(queueName); err != nil {
		if isAsynqQueueNotExistError(err) {
			return nil
		}
		return err
	}

	var firstErr error
	setErr := func(e error) {
		if e != nil && !isAsynqQueueNotExistError(e) && firstErr == nil {
			firstErr = e
		}
	}

	if _, err := q.inspector.DeleteAllPendingTasks(queueName); err != nil {
		setErr(err)
	}
	if _, err := q.inspector.DeleteAllScheduledTasks(queueName); err != nil {
		setErr(err)
	}
	if _, err := q.inspector.DeleteAllRetryTasks(queueName); err != nil {
		setErr(err)
	}
	if _, err := q.inspector.DeleteAllArchivedTasks(queueName); err != nil {
		setErr(err)
	}
	if _, err := q.inspector.DeleteAllCompletedTasks(queueName); err != nil {
		setErr(err)
	}

	active, err := q.inspector.ListActiveTasks(queueName)
	if err != nil {
		setErr(err)
	} else {
		for _, t := range active {
			setErr(q.inspector.CancelProcessing(t.ID))
		}
	}

	_ = ctx
	return firstErr
}

// Close closes the Asynq client and inspector.
func (q *AsynqQueue) Close() error {
	var errs []error
	if err := q.client.Close(); err != nil {
		errs = append(errs, err)
	}
	if err := q.inspector.Close(); err != nil {
		errs = append(errs, err)
	}
	return errors.Join(errs...)
}

// AsynqRedisConnOpt parses a redis:// URL for asynq.NewClient / NewServer / NewInspector.
func AsynqRedisConnOpt(connection string) (asynq.RedisConnOpt, error) {
	parsed, err := redis.ParseURL(connection)
	if err != nil {
		return asynq.RedisClientOpt{}, err
	}
	return asynq.RedisClientOpt{
		Addr:      parsed.Addr,
		Username:  parsed.Username,
		Password:  parsed.Password,
		DB:        parsed.DB,
		TLSConfig: parsed.TLSConfig,
	}, nil
}

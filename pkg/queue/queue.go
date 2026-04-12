package queue

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"
)

var (
	// ErrQueueEmpty 队列为空
	ErrQueueEmpty = errors.New("queue is empty")
	// ErrQueueFull 队列已满
	ErrQueueFull = errors.New("queue is full")
	// ErrQueueNotFound 队列不存在
	ErrQueueNotFound = errors.New("queue not found")
	// ErrJobNotFound 任务不存在
	ErrJobNotFound = errors.New("job not found")
	// ErrUnsupportedDriver 不支持的驱动类型
	ErrUnsupportedDriver = errors.New("unsupported queue driver")
	// ErrJobTimeout 任务超时
	ErrJobTimeout = errors.New("job timeout")
	// ErrMaxAttemptsExceeded 超过最大重试次数
	ErrMaxAttemptsExceeded = errors.New("max attempts exceeded")
	// ErrInvalidPayload 无效的任务数据
	ErrInvalidPayload = errors.New("invalid job payload")
	// ErrDuplicateJob 唯一任务已存在（相同队列 + unique key 尚在等待或执行中）
	ErrDuplicateJob = errors.New("duplicate job: unique key already queued or processing")
	// ErrPullNotSupported 当前驱动不支持 Pop / Delete / Release（Asynq 请使用 Server 消费）
	ErrPullNotSupported = errors.New("queue: Pop/Delete/Release not supported for this driver; use asynq.Server")
)

// NormalizeUniqueKey trims whitespace; empty string disables unique-queue deduplication.
func NormalizeUniqueKey(s string) string {
	return strings.TrimSpace(s)
}

// Job represents a queue job
type Job struct {
	ID        string          `json:"id"`
	Queue     string          `json:"queue"`
	Payload   json.RawMessage `json:"payload"`
	Attempts  int             `json:"attempts"`
	MaxRetry  int             `json:"max_retry"`
	Status    string          `json:"status"` // pending, processing, completed, failed
	Error     string          `json:"error,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	UpdatedAt time.Time       `json:"updated_at"`
}

// JobInterface 定义队列任务接口
type JobInterface interface {
	// Handle 处理任务
	Handle() error
	// GetQueue 获取队列名称
	GetQueue() string
	// GetAttempts 获取重试次数
	GetAttempts() int
	// GetMaxAttempts 获取最大重试次数
	GetMaxAttempts() int
	// GetDelay 获取延迟时间
	GetDelay() time.Duration
	// GetTimeout 获取超时时间
	GetTimeout() time.Duration
	// GetRetryAfter 获取重试等待时间
	GetRetryAfter() time.Duration
	// GetBackoff 获取退避策略
	GetBackoff() []time.Duration
	// GetPayload 获取任务数据
	GetPayload() []byte
	// SetPayload 设置任务数据
	SetPayload(payload []byte)
	// SetAttempts 设置重试次数
	SetAttempts(attempts int)
	// GetID 获取任务ID
	GetID() string
	// SetID 设置任务ID
	SetID(id string)
	// SetReservedAt 设置保留时间
	SetReservedAt(t *time.Time)
	// GetUniqueKey 非空时在同一队列内去重（Laravel ShouldBeUnique 风格）；空字符串表示不去重
	GetUniqueKey() string
	// TaskType Asynq 任务类型，与 JSON 字段 job_type 一致；空则入队时由 RegisterJobType 推断
	TaskType() string
}

// QueueInterface 定义队列驱动接口
type QueueInterface interface {
	// Push 推送任务到队列
	Push(ctx context.Context, job JobInterface) error
	// PushRaw 推送原始数据到队列
	PushRaw(ctx context.Context, queue string, payload []byte, options map[string]interface{}) error
	// Later 延迟推送任务
	Later(ctx context.Context, job JobInterface, delay time.Duration) error
	// Pop 从队列中取出任务
	Pop(ctx context.Context, queue string) (JobInterface, error)
	// Size 获取队列大小
	Size(ctx context.Context, queue string) (int64, error)
	// Delete 删除任务
	Delete(ctx context.Context, queue string, job JobInterface) error
	// Release 释放任务回队列
	Release(ctx context.Context, queue string, job JobInterface, delay time.Duration) error
	// Clear 清空队列
	Clear(ctx context.Context, queue string) error
	// Close 关闭队列连接
	Close() error
}

// Config represents queue configuration
type Config struct {
	Driver  string                 `mapstructure:"driver"`  // redis
	Options map[string]interface{} `mapstructure:"options"` // driver-specific options
}

// JobOption represents job options
type JobOption func(*jobOptions)

type jobOptions struct {
	Delay    time.Duration
	MaxRetry int
}

// WithDelay sets the delay for the job
func WithDelay(delay time.Duration) JobOption {
	return func(o *jobOptions) {
		o.Delay = delay
	}
}

// WithMaxRetry sets the maximum retry attempts for the job
func WithMaxRetry(maxRetry int) JobOption {
	return func(o *jobOptions) {
		o.MaxRetry = maxRetry
	}
}

var (
	drivers = make(map[string]func(config Config) (QueueInterface, error))
)

// Register registers a queue driver
func Register(name string, driver func(config Config) (QueueInterface, error)) {
	drivers[name] = driver
}

// New creates a new queue instance
func New(config Config) (QueueInterface, error) {
	driver, ok := drivers[config.Driver]
	if !ok {
		return nil, ErrUnsupportedDriver
	}
	return driver(config)
}

// Manager 队列管理器
type Manager struct {
	config Config
	driver QueueInterface
}

// NewManager 创建队列管理器
func NewManager(config Config) (*Manager, error) {
	var driver QueueInterface
	var err error

	switch strings.ToLower(strings.TrimSpace(config.Driver)) {
	case "redis", "asynq":
		driver, err = NewAsynqQueue(config)
	default:
		return nil, ErrUnsupportedDriver
	}

	if err != nil {
		return nil, err
	}

	return &Manager{
		config: config,
		driver: driver,
	}, nil
}

// Push 推送任务
func (m *Manager) Push(ctx context.Context, job JobInterface) error {
	ensureJobTypeForPush(job)
	return m.driver.Push(ctx, job)
}

// PushRaw 推送原始数据
func (m *Manager) PushRaw(ctx context.Context, queue string, payload []byte, options map[string]interface{}) error {
	return m.driver.PushRaw(ctx, queue, payload, options)
}

// Later 延迟推送
func (m *Manager) Later(ctx context.Context, job JobInterface, delay time.Duration) error {
	ensureJobTypeForPush(job)
	return m.driver.Later(ctx, job, delay)
}

// Pop 取出任务
func (m *Manager) Pop(ctx context.Context, queue string) (JobInterface, error) {
	return m.driver.Pop(ctx, queue)
}

// Size 获取队列大小
func (m *Manager) Size(ctx context.Context, queue string) (int64, error) {
	return m.driver.Size(ctx, queue)
}

// Delete 删除任务
func (m *Manager) Delete(ctx context.Context, queue string, job JobInterface) error {
	return m.driver.Delete(ctx, queue, job)
}

// Release 释放任务
func (m *Manager) Release(ctx context.Context, queue string, job JobInterface, delay time.Duration) error {
	return m.driver.Release(ctx, queue, job, delay)
}

// Clear 清空队列
func (m *Manager) Clear(ctx context.Context, queue string) error {
	return m.driver.Clear(ctx, queue)
}

// GetDriver 获取当前驱动
func (m *Manager) GetDriver() QueueInterface {
	return m.driver
}

// Close 关闭队列管理器
func (m *Manager) Close() error {
	return m.driver.Close()
}

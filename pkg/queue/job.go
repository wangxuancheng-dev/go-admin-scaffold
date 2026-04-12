package queue

import (
	"encoding/json"
	"time"
)

// BaseJob 基础任务结构体
type BaseJob struct {
	ID          string          `json:"id"`
	Queue       string          `json:"queue"`
	// UniqueKey 非空时，同一队列内相同 key 的任务在尚未完成前只会入队一次（需驱动支持）
	UniqueKey   string          `json:"unique_key,omitempty"`
	Payload     json.RawMessage `json:"payload"`
	Attempts    int             `json:"attempts"`
	MaxAttempts int             `json:"max_attempts"`
	Delay       time.Duration   `json:"delay"`
	Timeout     time.Duration   `json:"timeout"`
	RetryAfter  time.Duration   `json:"retry_after"`
	Backoff     []time.Duration `json:"backoff"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
	ReservedAt  *time.Time      `json:"reserved_at,omitempty"`
}

// NewBaseJob 创建基础任务
func NewBaseJob(queue string, payload interface{}, options map[string]interface{}) (*BaseJob, error) {
	// 序列化payload
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}

	job := &BaseJob{
		Queue:       queue,
		Payload:     payloadBytes,
		Attempts:    0,
		MaxAttempts: 3, // 默认最大重试3次
		Delay:       0,
		Timeout:     60 * time.Second,                                                        // 默认超时60秒
		RetryAfter:  60 * time.Second,                                                        // 默认重试等待60秒
		Backoff:     []time.Duration{60 * time.Second, 300 * time.Second, 900 * time.Second}, // 默认退避策略
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
	}

	// 应用选项
	if options != nil {
		if v, ok := options["max_attempts"].(int); ok {
			job.MaxAttempts = v
		}
		if v, ok := options["delay"].(time.Duration); ok {
			job.Delay = v
		}
		if v, ok := options["timeout"].(time.Duration); ok {
			job.Timeout = v
		}
		if v, ok := options["retry_after"].(time.Duration); ok {
			job.RetryAfter = v
		}
		if v, ok := options["backoff"].([]time.Duration); ok {
			job.Backoff = v
		}
		if v, ok := options["unique_key"].(string); ok {
			job.UniqueKey = v
		}
	}

	return job, nil
}

// Handle 处理任务
func (j *BaseJob) Handle() error {
	return nil
}

// GetQueue 获取队列名称
func (j *BaseJob) GetQueue() string {
	return j.Queue
}

// GetAttempts 获取重试次数
func (j *BaseJob) GetAttempts() int {
	return j.Attempts
}

// GetMaxAttempts 获取最大重试次数
func (j *BaseJob) GetMaxAttempts() int {
	return j.MaxAttempts
}

// GetDelay 获取延迟时间
func (j *BaseJob) GetDelay() time.Duration {
	return j.Delay
}

// GetTimeout 获取超时时间
func (j *BaseJob) GetTimeout() time.Duration {
	return j.Timeout
}

// GetRetryAfter 获取重试等待时间
func (j *BaseJob) GetRetryAfter() time.Duration {
	return j.RetryAfter
}

// GetBackoff 获取退避策略
func (j *BaseJob) GetBackoff() []time.Duration {
	return j.Backoff
}

// GetPayload 获取任务数据
func (j *BaseJob) GetPayload() []byte {
	return j.Payload
}

// SetPayload 设置任务数据
func (j *BaseJob) SetPayload(payload []byte) {
	j.Payload = payload
	j.UpdatedAt = time.Now()
}

// SetAttempts 设置重试次数
func (j *BaseJob) SetAttempts(attempts int) {
	j.Attempts = attempts
	j.UpdatedAt = time.Now()
}

// GetID 获取任务ID
func (j *BaseJob) GetID() string {
	return j.ID
}

// SetID 设置任务ID
func (j *BaseJob) SetID(id string) {
	j.ID = id
	j.UpdatedAt = time.Now()
}

// SetReservedAt 设置保留时间
func (j *BaseJob) SetReservedAt(t *time.Time) {
	j.ReservedAt = t
	j.UpdatedAt = time.Now()
}

// GetUniqueKey 返回唯一键（去重用）
func (j *BaseJob) GetUniqueKey() string {
	return j.UniqueKey
}

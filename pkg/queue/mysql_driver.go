package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// MySQLJob represents a job in the MySQL database
type MySQLJob struct {
	ID        string         `gorm:"primaryKey;type:varchar(36)"`
	Queue     string         `gorm:"type:varchar(100);uniqueIndex:uq_queue_jobs_uniq,priority:1"`
	UniqueKey *string        `gorm:"type:varchar(191);uniqueIndex:uq_queue_jobs_uniq,priority:2"`
	Payload   string         `gorm:"type:text"`
	Attempts  int            `gorm:"default:0"`
	MaxRetry  int            `gorm:"default:3"`
	Status    string         `gorm:"type:varchar(20);index"`
	Error     string         `gorm:"type:text"`
	CreatedAt time.Time      `gorm:"type:timestamp"`
	UpdatedAt time.Time      `gorm:"type:timestamp"`
	DeletedAt gorm.DeletedAt `gorm:"index;type:timestamp"`
}

// TableName returns the table name for the job
func (MySQLJob) TableName() string {
	return "queue_jobs"
}

func init() {
	Register("mysql", func(config Config) (QueueInterface, error) {
		return NewMySQLQueue(config)
	})
}

// MySQLQueue MySQL队列驱动
type MySQLQueue struct {
	db     *gorm.DB
	config Config
}

// NewMySQLQueue 创建MySQL队列驱动
func NewMySQLQueue(config Config) (*MySQLQueue, error) {
	db, ok := config.Options["db"].(*gorm.DB)
	if !ok {
		return nil, fmt.Errorf("mysql queue requires a *gorm.DB instance")
	}

	// Auto migrate the jobs table
	if err := db.AutoMigrate(&MySQLJob{}); err != nil {
		return nil, err
	}

	return &MySQLQueue{
		db:     db,
		config: config,
	}, nil
}

// Push 推送任务到队列
func (q *MySQLQueue) Push(ctx context.Context, job JobInterface) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}

	queue := job.GetQueue()
	if queue == "" {
		if queueOpt, ok := q.config.Options["queue"].(string); ok {
			queue = queueOpt
		} else {
			return fmt.Errorf("queue name not found in options")
		}
	}

	mysqlJob := &MySQLJob{
		ID:        job.GetID(),
		Queue:     queue,
		UniqueKey: dedupeKeyPtr(job.GetUniqueKey()),
		Payload:   string(payload),
		Attempts:  job.GetAttempts(),
		MaxRetry:  job.GetMaxAttempts(),
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if job.GetDelay() > 0 {
		mysqlJob.CreatedAt = time.Now().Add(job.GetDelay())
	}

	if err := q.db.WithContext(ctx).Create(mysqlJob).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrDuplicateJob
		}
		return err
	}
	return nil
}

// PushRaw 推送原始数据到队列
func (q *MySQLQueue) PushRaw(ctx context.Context, queue string, payload []byte, options map[string]interface{}) error {
	if queue == "" {
		if queueOpt, ok := q.config.Options["queue"].(string); ok {
			queue = queueOpt
		} else {
			return fmt.Errorf("queue name not found in options")
		}
	}

	delay := time.Duration(0)
	if v, ok := options["delay"].(time.Duration); ok {
		delay = v
	}

	maxAttempts := 3
	if v, ok := options["max_attempts"].(int); ok {
		maxAttempts = v
	}

	mysqlJob := &MySQLJob{
		ID:        uuid.New().String(),
		Queue:     queue,
		Payload:   string(payload),
		MaxRetry:  maxAttempts,
		Status:    "pending",
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if delay > 0 {
		mysqlJob.CreatedAt = time.Now().Add(delay)
	}

	uk := ""
	if v, ok := options["unique_key"].(string); ok {
		uk = v
	}
	payload = mergeUniqueKeyIntoPayload(payload, uk)

	mysqlJob.Payload = string(payload)
	mysqlJob.UniqueKey = dedupeKeyPtr(uk)

	if err := q.db.WithContext(ctx).Create(mysqlJob).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrDuplicateJob
		}
		return err
	}
	return nil
}

// Later 延迟推送任务
func (q *MySQLQueue) Later(ctx context.Context, job JobInterface, delay time.Duration) error {
	payload, err := json.Marshal(job)
	if err != nil {
		return err
	}

	queue := job.GetQueue()
	if queue == "" {
		if queueOpt, ok := q.config.Options["queue"].(string); ok {
			queue = queueOpt
		} else {
			return fmt.Errorf("queue name not found in options")
		}
	}

	mysqlJob := &MySQLJob{
		ID:        job.GetID(),
		Queue:     queue,
		UniqueKey: dedupeKeyPtr(job.GetUniqueKey()),
		Payload:   string(payload),
		Attempts:  job.GetAttempts(),
		MaxRetry:  job.GetMaxAttempts(),
		Status:    "pending",
		CreatedAt: time.Now().Add(delay),
		UpdatedAt: time.Now(),
	}

	if err := q.db.WithContext(ctx).Create(mysqlJob).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrDuplicateJob
		}
		return err
	}
	return nil
}

// Pop 从队列中取出任务
func (q *MySQLQueue) Pop(ctx context.Context, queue string) (JobInterface, error) {
	if queue == "" {
		if queueOpt, ok := q.config.Options["queue"].(string); ok {
			queue = queueOpt
		} else {
			return nil, fmt.Errorf("queue name not found in options")
		}
	}

	var mysqlJob MySQLJob

	err := q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Get the next available job (SKIP LOCKED: concurrent workers)
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("queue = ? AND status = ? AND created_at <= ?", queue, "pending", time.Now()).
			Order("created_at ASC").
			First(&mysqlJob).Error; err != nil {
			if err == gorm.ErrRecordNotFound {
				return ErrQueueEmpty
			}
			return err
		}

		// Update job status
		mysqlJob.Status = "processing"
		mysqlJob.Attempts++
		mysqlJob.UpdatedAt = time.Now()

		return tx.Save(&mysqlJob).Error
	})

	if err != nil {
		return nil, err
	}

	var job BaseJob
	if err := json.Unmarshal([]byte(mysqlJob.Payload), &job); err != nil {
		return nil, err
	}

	job.SetID(mysqlJob.ID)
	job.SetAttempts(mysqlJob.Attempts)

	return &job, nil
}

// Size 获取队列大小
func (q *MySQLQueue) Size(ctx context.Context, queue string) (int64, error) {
	if queue == "" {
		if queueOpt, ok := q.config.Options["queue"].(string); ok {
			queue = queueOpt
		} else {
			return 0, fmt.Errorf("queue name not found in options")
		}
	}

	var count int64
	err := q.db.WithContext(ctx).Model(&MySQLJob{}).
		Where("queue = ? AND status = ?", queue, "pending").
		Count(&count).Error
	return count, err
}

// Delete 删除任务
func (q *MySQLQueue) Delete(ctx context.Context, queue string, job JobInterface) error {
	if queue == "" {
		if queueOpt, ok := q.config.Options["queue"].(string); ok {
			queue = queueOpt
		} else {
			return fmt.Errorf("queue name not found in options")
		}
	}

	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&MySQLJob{}).Where("queue = ? AND id = ?", queue, job.GetID()).Update("unique_key", nil).Error; err != nil {
			return err
		}
		return tx.Where("queue = ? AND id = ?", queue, job.GetID()).Delete(&MySQLJob{}).Error
	})
}

// Release 释放任务回队列
func (q *MySQLQueue) Release(ctx context.Context, queue string, job JobInterface, delay time.Duration) error {
	if queue == "" {
		if queueOpt, ok := q.config.Options["queue"].(string); ok {
			queue = queueOpt
		} else {
			return fmt.Errorf("queue name not found in options")
		}
	}

	// 增加重试次数
	attempts := job.GetAttempts() + 1

	// 如果超过最大重试次数，直接删除
	if attempts >= job.GetMaxAttempts() {
		return q.Delete(ctx, queue, job)
	}

	updates := map[string]interface{}{
		"status":     "pending",
		"attempts":   attempts,
		"created_at": time.Now().Add(delay),
		"updated_at": time.Now(),
	}

	return q.db.WithContext(ctx).Model(&MySQLJob{}).
		Where("queue = ? AND id = ?", queue, job.GetID()).
		Updates(updates).Error
}

// Clear 清空队列
func (q *MySQLQueue) Clear(ctx context.Context, queue string) error {
	if queue == "" {
		if queueOpt, ok := q.config.Options["queue"].(string); ok {
			queue = queueOpt
		} else {
			return fmt.Errorf("queue name not found in options")
		}
	}

	return q.db.WithContext(ctx).Where("queue = ?", queue).Delete(&MySQLJob{}).Error
}

// Close 关闭队列连接
func (q *MySQLQueue) Close() error {
	sqlDB, err := q.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

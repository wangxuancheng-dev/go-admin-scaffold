package queue

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// DatabaseQueue 数据库队列驱动
type DatabaseQueue struct {
	db     *gorm.DB
	config Config
}

// QueueJob 数据库队列任务表结构
type QueueJob struct {
	ID          uint64         `gorm:"primaryKey"`
	Queue       string         `gorm:"index;not null;uniqueIndex:uq_queue_jobs_uniq,priority:1"`
	UniqueKey   *string        `gorm:"size:191;uniqueIndex:uq_queue_jobs_uniq,priority:2"`
	Payload     []byte         `gorm:"type:json;not null"`
	Attempts    int            `gorm:"default:0;not null"`
	MaxAttempts int            `gorm:"default:3;not null"`
	Delay       int64          `gorm:"default:0;not null"`
	Timeout     int64          `gorm:"default:60;not null"`
	RetryAfter  int64          `gorm:"default:60;not null"`
	Backoff     []byte         `gorm:"type:json"`
	CreatedAt   time.Time      `gorm:"type:timestamp;not null"`
	UpdatedAt   time.Time      `gorm:"type:timestamp;not null"`
	ReservedAt  *time.Time     `gorm:"type:timestamp;index"`
	AvailableAt time.Time      `gorm:"type:timestamp;index;not null"`
	DeletedAt   gorm.DeletedAt `gorm:"index;type:timestamp"`
}

// TableName 指定表名
func (QueueJob) TableName() string {
	return "queue_jobs"
}

func dedupeKeyPtr(raw string) *string {
	s := NormalizeUniqueKey(raw)
	if s == "" {
		return nil
	}
	return &s
}

// NewDatabaseQueue 创建数据库队列驱动
func NewDatabaseQueue(config Config) (*DatabaseQueue, error) {
	// 获取数据库连接
	db, ok := config.Options["db"].(*gorm.DB)
	if !ok {
		return nil, fmt.Errorf("database queue requires a *gorm.DB instance")
	}

	// 自动迁移表结构
	if err := db.AutoMigrate(&QueueJob{}); err != nil {
		return nil, fmt.Errorf("failed to migrate queue_jobs table: %v", err)
	}

	return &DatabaseQueue{
		db:     db,
		config: config,
	}, nil
}

// Push 推送任务到队列
func (q *DatabaseQueue) Push(ctx context.Context, job JobInterface) error {
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

	// 序列化退避策略
	backoff, err := json.Marshal(job.GetBackoff())
	if err != nil {
		return err
	}

	// 计算可用时间
	availableAt := time.Now()
	if job.GetDelay() > 0 {
		availableAt = availableAt.Add(job.GetDelay())
	}

	// 创建任务记录
	queueJob := &QueueJob{
		Queue:       queue,
		UniqueKey:   dedupeKeyPtr(job.GetUniqueKey()),
		Payload:     payload,
		Attempts:    job.GetAttempts(),
		MaxAttempts: job.GetMaxAttempts(),
		Delay:       int64(job.GetDelay().Seconds()),
		Timeout:     int64(job.GetTimeout().Seconds()),
		RetryAfter:  int64(job.GetRetryAfter().Seconds()),
		Backoff:     backoff,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		AvailableAt: availableAt,
	}

	if err := q.db.WithContext(ctx).Create(queueJob).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrDuplicateJob
		}
		return err
	}
	return nil
}

// PushRaw 推送原始数据到队列
func (q *DatabaseQueue) PushRaw(ctx context.Context, queue string, payload []byte, options map[string]interface{}) error {
	if queue == "" {
		if queueOpt, ok := q.config.Options["queue"].(string); ok {
			queue = queueOpt
		} else {
			return fmt.Errorf("queue name not found in options")
		}
	}

	// 解析选项
	delay := time.Duration(0)
	if v, ok := options["delay"].(time.Duration); ok {
		delay = v
	}

	maxAttempts := 3
	if v, ok := options["max_attempts"].(int); ok {
		maxAttempts = v
	}

	timeout := 60 * time.Second
	if v, ok := options["timeout"].(time.Duration); ok {
		timeout = v
	}

	retryAfter := 60 * time.Second
	if v, ok := options["retry_after"].(time.Duration); ok {
		retryAfter = v
	}

	backoff := []time.Duration{60 * time.Second, 300 * time.Second, 900 * time.Second}
	if v, ok := options["backoff"].([]time.Duration); ok {
		backoff = v
	}

	backoffBytes, err := json.Marshal(backoff)
	if err != nil {
		return err
	}

	uk := ""
	if v, ok := options["unique_key"].(string); ok {
		uk = v
	}
	payload = mergeUniqueKeyIntoPayload(payload, uk)

	// 计算可用时间
	availableAt := time.Now()
	if delay > 0 {
		availableAt = availableAt.Add(delay)
	}

	// 创建任务记录
	queueJob := &QueueJob{
		Queue:       queue,
		UniqueKey:   dedupeKeyPtr(uk),
		Payload:     payload,
		Attempts:    0,
		MaxAttempts: maxAttempts,
		Delay:       int64(delay.Seconds()),
		Timeout:     int64(timeout.Seconds()),
		RetryAfter:  int64(retryAfter.Seconds()),
		Backoff:     backoffBytes,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		AvailableAt: availableAt,
	}

	if err := q.db.WithContext(ctx).Create(queueJob).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrDuplicateJob
		}
		return err
	}
	return nil
}

// Later 延迟推送任务
func (q *DatabaseQueue) Later(ctx context.Context, job JobInterface, delay time.Duration) error {
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

	// 序列化退避策略
	backoff, err := json.Marshal(job.GetBackoff())
	if err != nil {
		return err
	}

	// 计算可用时间
	availableAt := time.Now().Add(delay)

	// 创建任务记录
	queueJob := &QueueJob{
		Queue:       queue,
		UniqueKey:   dedupeKeyPtr(job.GetUniqueKey()),
		Payload:     payload,
		Attempts:    job.GetAttempts(),
		MaxAttempts: job.GetMaxAttempts(),
		Delay:       int64(delay.Seconds()),
		Timeout:     int64(job.GetTimeout().Seconds()),
		RetryAfter:  int64(job.GetRetryAfter().Seconds()),
		Backoff:     backoff,
		CreatedAt:   time.Now(),
		UpdatedAt:   time.Now(),
		AvailableAt: availableAt,
	}

	if err := q.db.WithContext(ctx).Create(queueJob).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrDuplicateJob
		}
		return err
	}
	return nil
}

// Pop 从队列中取出任务
func (q *DatabaseQueue) Pop(ctx context.Context, queue string) (JobInterface, error) {
	if queue == "" {
		if queueOpt, ok := q.config.Options["queue"].(string); ok {
			queue = queueOpt
		} else {
			return nil, fmt.Errorf("queue name not found in options")
		}
	}

	// 开启事务
	tx := q.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
		}
	}()

	// 查找并锁定一个可用的任务（SKIP LOCKED 避免多 Worker 抢同一行）
	var queueJob QueueJob
	err := tx.Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
		Where("queue = ? AND reserved_at IS NULL AND available_at <= ?", queue, time.Now()).
		Order("available_at ASC").
		First(&queueJob).Error

	if err == gorm.ErrRecordNotFound {
		tx.Rollback()
		return nil, ErrQueueEmpty
	}
	if err != nil {
		tx.Rollback()
		return nil, err
	}

	// 标记任务为已保留
	now := time.Now()
	queueJob.ReservedAt = &now
	queueJob.UpdatedAt = now

	if err := tx.Save(&queueJob).Error; err != nil {
		tx.Rollback()
		return nil, err
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		return nil, err
	}

	// 转换为Job接口
	var job BaseJob
	if err := json.Unmarshal(queueJob.Payload, &job); err != nil {
		return nil, err
	}

	// 设置任务ID和保留时间
	job.SetID(fmt.Sprintf("%d", queueJob.ID))
	job.SetReservedAt(queueJob.ReservedAt)
	job.SetAttempts(queueJob.Attempts)

	return &job, nil
}

// Size 获取队列大小
func (q *DatabaseQueue) Size(ctx context.Context, queue string) (int64, error) {
	if queue == "" {
		if queueOpt, ok := q.config.Options["queue"].(string); ok {
			queue = queueOpt
		} else {
			return 0, fmt.Errorf("queue name not found in options")
		}
	}

	var count int64
	err := q.db.WithContext(ctx).Model(&QueueJob{}).
		Where("queue = ? AND reserved_at IS NULL", queue).
		Count(&count).Error

	return count, err
}

// Delete 删除任务
func (q *DatabaseQueue) Delete(ctx context.Context, queue string, job JobInterface) error {
	if queue == "" {
		if queueOpt, ok := q.config.Options["queue"].(string); ok {
			queue = queueOpt
		} else {
			return fmt.Errorf("queue name not found in options")
		}
	}

	return q.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&QueueJob{}).Where("queue = ? AND id = ?", queue, job.GetID()).Update("unique_key", nil).Error; err != nil {
			return err
		}
		return tx.Where("queue = ? AND id = ?", queue, job.GetID()).Delete(&QueueJob{}).Error
	})
}

// Release 释放任务回队列
func (q *DatabaseQueue) Release(ctx context.Context, queue string, job JobInterface, delay time.Duration) error {
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

	// 计算新的可用时间
	availableAt := time.Now()
	if delay > 0 {
		availableAt = availableAt.Add(delay)
	} else {
		// 使用退避策略
		backoff := job.GetBackoff()
		if len(backoff) > 0 {
			attempt := attempts - 1
			if attempt < len(backoff) {
				availableAt = availableAt.Add(backoff[attempt])
			} else {
				availableAt = availableAt.Add(backoff[len(backoff)-1])
			}
		} else {
			availableAt = availableAt.Add(job.GetRetryAfter())
		}
	}

	// 更新任务
	return q.db.WithContext(ctx).Model(&QueueJob{}).
		Where("queue = ? AND id = ?", queue, job.GetID()).
		Updates(map[string]interface{}{
			"attempts":     attempts,
			"reserved_at":  nil,
			"available_at": availableAt,
			"updated_at":   time.Now(),
		}).Error
}

// Clear 清空队列
func (q *DatabaseQueue) Clear(ctx context.Context, queue string) error {
	if queue == "" {
		if queueOpt, ok := q.config.Options["queue"].(string); ok {
			queue = queueOpt
		} else {
			return fmt.Errorf("queue name not found in options")
		}
	}

	return q.db.WithContext(ctx).Where("queue = ?", queue).Delete(&QueueJob{}).Error
}

// Close 关闭队列连接
func (q *DatabaseQueue) Close() error {
	sqlDB, err := q.db.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

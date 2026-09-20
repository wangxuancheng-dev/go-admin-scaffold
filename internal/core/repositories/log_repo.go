package repositories

import (
	"context"

	"go-admin-scaffold/internal/core/models"

	"gorm.io/gorm"
)

type LogRepository struct {
	*BaseRepository
}

func NewLogRepository(db *gorm.DB) *LogRepository {
	return &LogRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// CreateLoginLog creates a new login log record
func (r *LogRepository) CreateLoginLog(ctx context.Context, log *models.LoginLog) error {
	return r.Create(ctx, log)
}

// CreateOperationLog creates a new operation log record
func (r *LogRepository) CreateOperationLog(ctx context.Context, log *models.OperationLog) error {
	return r.Create(ctx, log)
}

// ListLoginLogs retrieves a paginated list of login logs
func (r *LogRepository) ListLoginLogs(ctx context.Context, pagination *models.Pagination, query map[string]interface{}) ([]models.LoginLog, error) {
	var logs []models.LoginLog
	db := r.db.WithContext(ctx)

	// Apply query conditions
	for key, value := range query {
		if value != nil && value != "" {
			db = db.Where(key, value)
		}
	}

	// Get total count
	if err := db.Model(&models.LoginLog{}).Count(&pagination.Total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	err := db.Order("login_time DESC").
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Find(&logs).Error

	if err != nil {
		return nil, err
	}

	return logs, nil
}

// ListOperationLogs retrieves a paginated list of operation logs
func (r *LogRepository) ListOperationLogs(ctx context.Context, pagination *models.Pagination, query map[string]interface{}) ([]models.OperationLog, error) {
	var logs []models.OperationLog
	db := r.db.WithContext(ctx)

	// Apply query conditions
	for key, value := range query {
		if value != nil && value != "" {
			db = db.Where(key, value)
		}
	}

	// Get total count
	if err := db.Model(&models.OperationLog{}).Count(&pagination.Total).Error; err != nil {
		return nil, err
	}

	// Get paginated results
	err := db.Order("operation_time DESC").
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Find(&logs).Error

	if err != nil {
		return nil, err
	}

	return logs, nil
}

// GetLoginLogsByUserID retrieves login logs for a specific user
func (r *LogRepository) GetLoginLogsByUserID(ctx context.Context, userID uint, limit int) ([]models.LoginLog, error) {
	var logs []models.LoginLog
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("login_time DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

// GetOperationLogsByUserID retrieves operation logs for a specific user
func (r *LogRepository) GetOperationLogsByUserID(ctx context.Context, userID uint, limit int) ([]models.OperationLog, error) {
	var logs []models.OperationLog
	err := r.db.WithContext(ctx).
		Where("user_id = ?", userID).
		Order("operation_time DESC").
		Limit(limit).
		Find(&logs).Error
	return logs, err
}

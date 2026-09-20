package services

import (
	"context"
	"time"

	"go-admin-scaffold/internal/core/models"
)

type LogService struct {
	logRepo LogRepository
}

func NewLogService(logRepo LogRepository) *LogService {
	return &LogService{
		logRepo: logRepo,
	}
}

// RecordLoginLog records a login attempt
func (s *LogService) RecordLoginLog(ctx context.Context, userID uint, username, ip, userAgent string, status int, message string) error {
	log := &models.LoginLog{
		UserID:    userID,
		Username:  username,
		IP:        ip,
		UserAgent: userAgent,
		Status:    status,
		Message:   message,
		LoginTime: models.CustomTime(time.Now()),
	}
	return s.logRepo.CreateLoginLog(ctx, log)
}

// RecordOperationLog records a user operation
func (s *LogService) RecordOperationLog(ctx context.Context, log *models.OperationLog) error {
	log.OperationTime = models.CustomTime(time.Now())
	return s.logRepo.CreateOperationLog(ctx, log)
}

type LogQuery struct {
	Username   string    `form:"username"`
	IP         string    `form:"ip"`
	Status     *int      `form:"status"`
	StartTime  time.Time `form:"start_time"`
	EndTime    time.Time `form:"end_time"`
	Module     string    `form:"module"`
	Action     string    `form:"action"`
	BusinessID string    `form:"business_id"`
}

// ListLoginLogs retrieves a paginated list of login logs
func (s *LogService) ListLoginLogs(ctx context.Context, pagination *models.Pagination, query *LogQuery) ([]models.LoginLog, error) {
	conditions := make(map[string]interface{})

	if query != nil {
		if query.Username != "" {
			conditions["username"] = query.Username
		}
		if query.IP != "" {
			conditions["ip"] = query.IP
		}
		if query.Status != nil {
			conditions["status"] = *query.Status
		}
		if !query.StartTime.IsZero() && !query.EndTime.IsZero() {
			conditions["login_time BETWEEN ? AND ?"] = []time.Time{query.StartTime, query.EndTime}
		}
	}

	return s.logRepo.ListLoginLogs(ctx, pagination, conditions)
}

// ListOperationLogs retrieves a paginated list of operation logs
func (s *LogService) ListOperationLogs(ctx context.Context, pagination *models.Pagination, query *LogQuery) ([]models.OperationLog, error) {
	conditions := make(map[string]interface{})

	if query != nil {
		if query.Username != "" {
			conditions["username"] = query.Username
		}
		if query.IP != "" {
			conditions["ip"] = query.IP
		}
		if query.Status != nil {
			conditions["status"] = *query.Status
		}
		if query.Module != "" {
			conditions["module"] = query.Module
		}
		if query.Action != "" {
			conditions["action"] = query.Action
		}
		if query.BusinessID != "" {
			conditions["business_id"] = query.BusinessID
		}
		if !query.StartTime.IsZero() && !query.EndTime.IsZero() {
			conditions["operation_time BETWEEN ? AND ?"] = []time.Time{query.StartTime, query.EndTime}
		}
	}

	return s.logRepo.ListOperationLogs(ctx, pagination, conditions)
}

// GetUserLoginHistory retrieves recent login history for a user
func (s *LogService) GetUserLoginHistory(ctx context.Context, userID uint, limit int) ([]models.LoginLog, error) {
	return s.logRepo.GetLoginLogsByUserID(ctx, userID, limit)
}

// GetUserOperationHistory retrieves recent operations for a user
func (s *LogService) GetUserOperationHistory(ctx context.Context, userID uint, limit int) ([]models.OperationLog, error) {
	return s.logRepo.GetOperationLogsByUserID(ctx, userID, limit)
}

// GetLoginLogs retrieves a paginated list of login logs
func (s *LogService) GetLoginLogs(ctx context.Context, page, pageSize int) ([]models.LoginLog, int64, error) {
	pagination := &models.Pagination{
		Page:     page,
		PageSize: pageSize,
	}
	logs, err := s.logRepo.ListLoginLogs(ctx, pagination, nil)
	if err != nil {
		return nil, 0, err
	}
	return logs, pagination.Total, nil
}

// GetOperationLogs retrieves a paginated list of operation logs
func (s *LogService) GetOperationLogs(ctx context.Context, page, pageSize int) ([]models.OperationLog, int64, error) {
	pagination := &models.Pagination{
		Page:     page,
		PageSize: pageSize,
	}
	logs, err := s.logRepo.ListOperationLogs(ctx, pagination, nil)
	if err != nil {
		return nil, 0, err
	}
	return logs, pagination.Total, nil
}

// GetUserLoginLogs retrieves login logs for a specific user
func (s *LogService) GetUserLoginLogs(ctx context.Context, userID uint) ([]models.LoginLog, error) {
	conditions := map[string]interface{}{
		"user_id": userID,
	}
	return s.logRepo.ListLoginLogs(ctx, &models.Pagination{Page: 1, PageSize: 100}, conditions)
}

// GetUserOperationLogs retrieves operation logs for a specific user
func (s *LogService) GetUserOperationLogs(ctx context.Context, userID uint) ([]models.OperationLog, error) {
	conditions := map[string]interface{}{
		"user_id": userID,
	}
	return s.logRepo.ListOperationLogs(ctx, &models.Pagination{Page: 1, PageSize: 100}, conditions)
}

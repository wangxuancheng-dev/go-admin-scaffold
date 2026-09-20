package services

import (
	"context"
	"testing"

	"go-admin-scaffold/internal/core/models"

	"github.com/stretchr/testify/require"
)

type stubLogRepo struct {
	logins []models.LoginLog
	ops    []models.OperationLog
}

func (s *stubLogRepo) CreateLoginLog(ctx context.Context, log *models.LoginLog) error {
	s.logins = append(s.logins, *log)
	return nil
}

func (s *stubLogRepo) CreateOperationLog(ctx context.Context, log *models.OperationLog) error {
	s.ops = append(s.ops, *log)
	return nil
}

func (s *stubLogRepo) ListLoginLogs(ctx context.Context, pagination *models.Pagination, query map[string]interface{}) ([]models.LoginLog, error) {
	pagination.Total = int64(len(s.logins))
	return s.logins, nil
}

func (s *stubLogRepo) ListOperationLogs(ctx context.Context, pagination *models.Pagination, query map[string]interface{}) ([]models.OperationLog, error) {
	pagination.Total = int64(len(s.ops))
	return s.ops, nil
}

func (s *stubLogRepo) GetLoginLogsByUserID(ctx context.Context, userID uint, limit int) ([]models.LoginLog, error) {
	return s.logins, nil
}

func (s *stubLogRepo) GetOperationLogsByUserID(ctx context.Context, userID uint, limit int) ([]models.OperationLog, error) {
	return s.ops, nil
}

func TestLogService_recordAndList(t *testing.T) {
	repo := &stubLogRepo{}
	svc := NewLogService(repo)
	ctx := context.Background()

	require.NoError(t, svc.RecordLoginLog(ctx, 1, "admin", "127.0.0.1", "ua", 1, "ok"))
	require.NoError(t, svc.RecordOperationLog(ctx, &models.OperationLog{UserID: 1, Username: "admin", Module: "user", Action: "create"}))

	status := 1
	logins, err := svc.ListLoginLogs(ctx, &models.Pagination{Page: 1, PageSize: 10}, &LogQuery{Username: "admin", Status: &status})
	require.NoError(t, err)
	require.Len(t, logins, 1)

	ops, err := svc.ListOperationLogs(ctx, &models.Pagination{Page: 1, PageSize: 10}, &LogQuery{Module: "user"})
	require.NoError(t, err)
	require.Len(t, ops, 1)

	hist, err := svc.GetUserLoginHistory(ctx, 1, 10)
	require.NoError(t, err)
	require.Len(t, hist, 1)

	opHist, err := svc.GetUserOperationHistory(ctx, 1, 10)
	require.NoError(t, err)
	require.Len(t, opHist, 1)

	pagedLogins, total, err := svc.GetLoginLogs(ctx, 1, 10)
	require.NoError(t, err)
	require.Len(t, pagedLogins, 1)
	require.Equal(t, int64(1), total)

	pagedOps, opTotal, err := svc.GetOperationLogs(ctx, 1, 10)
	require.NoError(t, err)
	require.Len(t, pagedOps, 1)
	require.Equal(t, int64(1), opTotal)

	userLogins, err := svc.GetUserLoginLogs(ctx, 1)
	require.NoError(t, err)
	require.Len(t, userLogins, 1)

	userOps, err := svc.GetUserOperationLogs(ctx, 1)
	require.NoError(t, err)
	require.Len(t, userOps, 1)
}

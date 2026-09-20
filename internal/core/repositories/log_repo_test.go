package repositories_test

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/repositories"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func setupLogRepoDB(t *testing.T) *gorm.DB {
	t.Helper()
	name := strings.ReplaceAll(t.Name(), "/", "_")
	db, err := gorm.Open(sqlite.Open(fmt.Sprintf("file:%s?mode=memory&cache=private", name)), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.LoginLog{}, &models.OperationLog{}))
	return db
}

func TestLogRepository_logs(t *testing.T) {
	db := setupLogRepoDB(t)

	repo := repositories.NewLogRepository(db)
	ctx := context.Background()
	now := models.CustomTime(time.Now())

	require.NoError(t, repo.CreateLoginLog(ctx, &models.LoginLog{
		UserID: 1, Username: "admin", Status: 1, Message: "ok", LoginTime: now,
	}))
	require.NoError(t, repo.CreateOperationLog(ctx, &models.OperationLog{
		UserID: 1, Username: "admin", Module: "user", Action: "create", Status: 1, OperationTime: now,
	}))

	page := &models.Pagination{Page: 1, PageSize: 10}
	logins, err := repo.ListLoginLogs(ctx, page, map[string]interface{}{"username": "admin"})
	require.NoError(t, err)
	require.Len(t, logins, 1)
	require.Equal(t, int64(1), page.Total)

	ops, err := repo.ListOperationLogs(ctx, page, map[string]interface{}{"module": "user"})
	require.NoError(t, err)
	require.Len(t, ops, 1)

	byUser, err := repo.GetLoginLogsByUserID(ctx, 1, 5)
	require.NoError(t, err)
	require.Len(t, byUser, 1)

	opByUser, err := repo.GetOperationLogsByUserID(ctx, 1, 5)
	require.NoError(t, err)
	require.Len(t, opByUser, 1)
}

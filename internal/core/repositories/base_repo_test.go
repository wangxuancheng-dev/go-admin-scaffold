package repositories_test

import (
	"context"
	"testing"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/repositories"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestBaseRepository_CRUD(t *testing.T) {
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Todo{}))

	repo := repositories.NewBaseRepository(db)
	ctx := context.Background()

	todo := &models.Todo{Title: "base"}
	require.NoError(t, repo.Create(ctx, todo))
	require.NotZero(t, todo.ID)

	var found models.Todo
	require.NoError(t, repo.FindByID(ctx, todo.ID, &found))
	require.Equal(t, "base", found.Title)

	found.Title = "updated"
	require.NoError(t, repo.Update(ctx, &found))

	page := &models.Pagination{Page: 1, PageSize: 10}
	var list []models.Todo
	require.NoError(t, repo.List(ctx, page, &list))
	require.Equal(t, int64(1), page.Total)
	require.Len(t, list, 1)

	require.NoError(t, repo.Transaction(ctx, func(tx *gorm.DB) error {
		return tx.Model(&models.Todo{}).Where("id = ?", todo.ID).Update("completed", true).Error
	}))

	require.NoError(t, repo.Delete(ctx, &models.Todo{BaseModel: models.BaseModel{ID: todo.ID}}))
}

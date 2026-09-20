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

func setupTodoDB(t *testing.T) *gorm.DB {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.Todo{}))
	return db
}

func TestTodoRepository_CRUD(t *testing.T) {
	db := setupTodoDB(t)
	repo := repositories.NewTodoRepository(db)
	ctx := context.Background()

	todo := &models.Todo{Title: "t1", Description: "d1"}
	require.NoError(t, repo.Create(ctx, todo))
	require.NotZero(t, todo.ID)

	got, err := repo.GetByID(ctx, todo.ID)
	require.NoError(t, err)
	require.Equal(t, "t1", got.Title)

	got.Completed = true
	require.NoError(t, repo.Update(ctx, got))

	page := &models.Pagination{Page: 1, PageSize: 10}
	list, err := repo.List(ctx, page)
	require.NoError(t, err)
	require.Len(t, list, 1)
	require.Equal(t, int64(1), page.Total)
	require.True(t, list[0].Completed)

	require.NoError(t, repo.Delete(ctx, todo.ID))
	_, err = repo.GetByID(ctx, todo.ID)
	require.Error(t, err)
}

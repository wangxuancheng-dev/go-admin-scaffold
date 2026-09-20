package services

import (
	"context"
	"testing"

	"go-admin-scaffold/internal/core/models"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

type stubTodoRepo struct {
	items map[uint]*models.Todo
	next  uint
}

func newStubTodoRepo() *stubTodoRepo {
	return &stubTodoRepo{items: make(map[uint]*models.Todo), next: 1}
}

func (s *stubTodoRepo) Create(ctx context.Context, todo *models.Todo) error {
	todo.ID = s.next
	s.next++
	cp := *todo
	s.items[todo.ID] = &cp
	return nil
}

func (s *stubTodoRepo) List(ctx context.Context, pagination *models.Pagination) ([]models.Todo, error) {
	out := make([]models.Todo, 0, len(s.items))
	for _, t := range s.items {
		out = append(out, *t)
	}
	pagination.Total = int64(len(out))
	return out, nil
}

func (s *stubTodoRepo) GetByID(ctx context.Context, id uint) (*models.Todo, error) {
	t, ok := s.items[id]
	if !ok {
		return nil, gorm.ErrRecordNotFound
	}
	cp := *t
	return &cp, nil
}

func (s *stubTodoRepo) Update(ctx context.Context, todo *models.Todo) error {
	if _, ok := s.items[todo.ID]; !ok {
		return gorm.ErrRecordNotFound
	}
	cp := *todo
	s.items[todo.ID] = &cp
	return nil
}

func (s *stubTodoRepo) Delete(ctx context.Context, id uint) error {
	delete(s.items, id)
	return nil
}

func TestTodoService_CRUD(t *testing.T) {
	svc := NewTodoService(newStubTodoRepo())
	ctx := context.Background()

	created, err := svc.Create(ctx, &CreateTodoRequest{Title: "a", Description: "d"})
	require.NoError(t, err)
	require.Equal(t, uint(1), created.ID)

	got, err := svc.GetByID(ctx, created.ID)
	require.NoError(t, err)
	require.Equal(t, "a", got.Title)

	updated, err := svc.Update(ctx, created.ID, &UpdateTodoRequest{Title: "b", Description: "e", Completed: true})
	require.NoError(t, err)
	require.True(t, updated.Completed)
	require.Equal(t, "b", updated.Title)

	list, err := svc.List(ctx, &models.Pagination{Page: 1, PageSize: 10})
	require.NoError(t, err)
	require.Len(t, list, 1)

	require.NoError(t, svc.Delete(ctx, created.ID))
	_, err = svc.GetByID(ctx, created.ID)
	require.ErrorIs(t, err, gorm.ErrRecordNotFound)
}

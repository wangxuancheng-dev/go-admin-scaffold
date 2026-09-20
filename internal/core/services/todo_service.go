package services

import (
	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/repositories"
	"context"
)

type TodoService struct {
	repo *repositories.TodoRepository
}

func NewTodoService(repo *repositories.TodoRepository) *TodoService {
	return &TodoService{repo: repo}
}

type CreateTodoRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
}

type UpdateTodoRequest struct {
	Title       string `json:"title" binding:"required"`
	Description string `json:"description"`
	Completed   bool   `json:"completed"`
}

func (s *TodoService) Create(ctx context.Context, req *CreateTodoRequest) (*models.Todo, error) {
	todo := &models.Todo{
		Title:       req.Title,
		Description: req.Description,
	}
	if err := s.repo.Create(ctx, todo); err != nil {
		return nil, err
	}
	return todo, nil
}

func (s *TodoService) List(ctx context.Context, pagination *models.Pagination) ([]models.Todo, error) {
	return s.repo.List(ctx, pagination)
}

func (s *TodoService) GetByID(ctx context.Context, id uint) (*models.Todo, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *TodoService) Update(ctx context.Context, id uint, req *UpdateTodoRequest) (*models.Todo, error) {
	todo, err := s.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}

	todo.Title = req.Title
	todo.Description = req.Description
	todo.Completed = req.Completed

	if err := s.repo.Update(ctx, todo); err != nil {
		return nil, err
	}
	return todo, nil
}

func (s *TodoService) Delete(ctx context.Context, id uint) error {
	return s.repo.Delete(ctx, id)
}

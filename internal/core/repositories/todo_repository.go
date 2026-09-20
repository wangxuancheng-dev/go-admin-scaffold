package repositories

import (
	"go-admin-scaffold/internal/core/models"
	"context"

	"gorm.io/gorm"
)

type TodoRepository struct {
	db *gorm.DB
}

func NewTodoRepository(db *gorm.DB) *TodoRepository {
	return &TodoRepository{db: db}
}

func (r *TodoRepository) Create(ctx context.Context, todo *models.Todo) error {
	return r.db.Create(todo).Error
}

func (r *TodoRepository) List(ctx context.Context, pagination *models.Pagination) ([]models.Todo, error) {
	var todos []models.Todo
	query := r.db.Model(&models.Todo{})

	if err := query.Count(&pagination.Total).Error; err != nil {
		return nil, err
	}

	if err := query.Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Order("created_at DESC").
		Find(&todos).Error; err != nil {
		return nil, err
	}

	return todos, nil
}

func (r *TodoRepository) GetByID(ctx context.Context, id uint) (*models.Todo, error) {
	var todo models.Todo
	if err := r.db.First(&todo, id).Error; err != nil {
		return nil, err
	}
	return &todo, nil
}

func (r *TodoRepository) Update(ctx context.Context, todo *models.Todo) error {
	return r.db.Save(todo).Error
}

func (r *TodoRepository) Delete(ctx context.Context, id uint) error {
	return r.db.Delete(&models.Todo{}, id).Error
}

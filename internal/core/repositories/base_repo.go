package repositories

import (
	"context"

	"go-admin-scaffold/internal/core/models"

	"gorm.io/gorm"
)

type BaseRepository struct {
	db *gorm.DB
}

func NewBaseRepository(db *gorm.DB) *BaseRepository {
	return &BaseRepository{db: db}
}

// Create inserts a new record into the database
func (r *BaseRepository) Create(ctx context.Context, model interface{}) error {
	return r.db.WithContext(ctx).Create(model).Error
}

// Update updates an existing record in the database
func (r *BaseRepository) Update(ctx context.Context, model interface{}) error {
	return r.db.WithContext(ctx).Save(model).Error
}

// Delete soft deletes a record from the database
func (r *BaseRepository) Delete(ctx context.Context, model interface{}) error {
	return r.db.WithContext(ctx).Delete(model).Error
}

// FindByID retrieves a record by its ID
func (r *BaseRepository) FindByID(ctx context.Context, id uint, model interface{}) error {
	return r.db.WithContext(ctx).First(model, id).Error
}

// List retrieves a paginated list of records
func (r *BaseRepository) List(ctx context.Context, pagination *models.Pagination, model interface{}, conditions ...interface{}) error {
	db := r.db.WithContext(ctx)

	// Apply conditions if any
	if len(conditions) > 0 {
		db = db.Where(conditions[0], conditions[1:]...)
	}

	// Get total count
	if err := db.Model(model).Count(&pagination.Total).Error; err != nil {
		return err
	}

	// Get paginated results
	return db.Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Find(model).Error
}

// Transaction executes operations within a database transaction
func (r *BaseRepository) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return r.db.WithContext(ctx).Transaction(fn)
}

package repositories

import (
	"context"

	"app/internal/config"
	"app/internal/core/models"
	"app/internal/core/types"
	"app/pkg/logger"

	"gorm.io/gorm"
)

type UserRepository struct {
	*BaseRepository
	config *config.Config
}

func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		BaseRepository: NewBaseRepository(db),
	}
}

// SetConfig sets the config instance
func (r *UserRepository) SetConfig(config *config.Config) {
	r.config = config
}

// FindByUsername retrieves a user by username
func (r *UserRepository) FindByUsername(ctx context.Context, username string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("username = ?", username).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// FindByEmail retrieves a user by email
func (r *UserRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Where("email = ?", email).First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// ListWithRoles retrieves a paginated list of users with their roles
func (r *UserRepository) ListWithRoles(ctx context.Context, pagination *models.Pagination) ([]models.User, error) {
	var users []models.User
	err := r.db.WithContext(ctx).
		Preload("Roles").
		Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	// Get total count
	if err := r.db.Model(&models.User{}).Count(&pagination.Total).Error; err != nil {
		return nil, err
	}

	return users, nil
}

// UpdateLastLogin updates the user's last login timestamp
func (r *UserRepository) UpdateLastLogin(ctx context.Context, userID uint) error {
	return r.db.WithContext(ctx).
		Model(&models.User{}).
		Where("id = ?", userID).
		UpdateColumn("last_login_at", gorm.Expr("NOW()")).
		Error
}

// FindByID retrieves a user by ID
func (r *UserRepository) FindByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Preload("Roles", func(db *gorm.DB) *gorm.DB {
			// Only load active roles
			return db.Where("status = ?", 1).Distinct()
		}).
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		logger.Error(ctx, "user FindByID failed", "error", err, "id", id)
		return nil, err
	}

	if r.config != nil {
		for _, adminID := range r.config.SuperAdminUintIDs() {
			if adminID == user.ID {
				user.IsSuperAdmin = true
				break
			}
		}
	}

	return &user, nil
}

// GetDB returns the database instance
func (r *UserRepository) GetDB() *gorm.DB {
	return r.db
}

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// Update updates an existing user
func (r *UserRepository) Update(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Save(user).Error
}

// Delete deletes a user by ID
func (r *UserRepository) Delete(ctx context.Context, id uint) error {
	return r.db.WithContext(ctx).Delete(&models.User{}, id).Error
}

// ListWithFilters retrieves a paginated list of users with search filters
func (r *UserRepository) ListWithFilters(ctx context.Context, pagination *models.Pagination, filters *types.UserSearchFilters) ([]models.User, error) {
	var users []models.User

	// Build the base query without joins first
	baseQuery := r.db.WithContext(ctx).Model(&models.User{})

	// Apply basic filters
	if filters != nil {
		if filters.Username != "" {
			baseQuery = baseQuery.Where("username LIKE ?", "%"+filters.Username+"%")
		}
		if filters.Email != "" {
			baseQuery = baseQuery.Where("email LIKE ?", "%"+filters.Email+"%")
		}
		if filters.Status != nil {
			baseQuery = baseQuery.Where("status = ?", *filters.Status)
		}
	}

	// Handle role filter separately to avoid JOIN conflicts with Preload
	var userIDs []uint
	if filters != nil && filters.RoleID > 0 {
		// First, get user IDs that have the specified role
		err := r.db.WithContext(ctx).
			Table("user_roles").
			Where("role_id = ?", filters.RoleID).
			Distinct("user_id"). // Ensure distinct user IDs
			Pluck("user_id", &userIDs).Error
		if err != nil {
			return nil, err
		}

		if len(userIDs) == 0 {
			// No users have this role, return empty result
			pagination.Total = 0
			return []models.User{}, nil
		}

		// Filter by these user IDs
		baseQuery = baseQuery.Where("id IN ?", userIDs)
	}

	// Get total count for pagination
	var total int64
	if err := baseQuery.Count(&total).Error; err != nil {
		return nil, err
	}
	pagination.Total = total

	// Now build the final query with Preload for roles
	finalQuery := r.db.WithContext(ctx).
		Preload("Roles", func(db *gorm.DB) *gorm.DB {
			// Ensure distinct roles to prevent duplicates
			return db.Distinct()
		})

	// Apply the same filters to the final query
	if filters != nil {
		if filters.Username != "" {
			finalQuery = finalQuery.Where("username LIKE ?", "%"+filters.Username+"%")
		}
		if filters.Email != "" {
			finalQuery = finalQuery.Where("email LIKE ?", "%"+filters.Email+"%")
		}
		if filters.Status != nil {
			finalQuery = finalQuery.Where("status = ?", *filters.Status)
		}
		if filters.RoleID > 0 && len(userIDs) > 0 {
			finalQuery = finalQuery.Where("id IN ?", userIDs)
		}
	}

	// Apply pagination and get results
	err := finalQuery.Offset(pagination.GetOffset()).
		Limit(pagination.GetLimit()).
		Find(&users).Error
	if err != nil {
		return nil, err
	}

	return users, nil
}

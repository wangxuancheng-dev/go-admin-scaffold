package repositories

import (
	"context"
	"errors"

	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/types"
	"go-admin-scaffold/pkg/logger"

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

// Create creates a new user
func (r *UserRepository) Create(ctx context.Context, user *models.User) error {
	return r.db.WithContext(ctx).Create(user).Error
}

// CreateWithRoles creates a user and assigns roles in a transaction.
// If roleIDs is empty, the default "user" role is assigned. Roles are preloaded onto user.
func (r *UserRepository) CreateWithRoles(ctx context.Context, user *models.User, roleIDs []uint) error {
	return r.Transaction(ctx, func(tx *gorm.DB) error {
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		ids := roleIDs
		if len(ids) == 0 {
			var userRole models.Role
			if err := tx.Where("code = ?", "user").First(&userRole).Error; err != nil {
				return err
			}
			ids = []uint{userRole.ID}
		}

		userRoles := make([]models.UserRole, 0, len(ids))
		for _, roleID := range ids {
			userRoles = append(userRoles, models.UserRole{
				UserID: user.ID,
				RoleID: roleID,
			})
		}
		if err := tx.Create(&userRoles).Error; err != nil {
			return err
		}

		return tx.Preload("Roles").First(user, user.ID).Error
	})
}

// FindBasicByID retrieves a user by ID without preloading associations
func (r *UserRepository) FindBasicByID(ctx context.Context, id uint) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).
		Select("id", "username", "email", "nickname", "avatar", "status").
		Where("id = ?", id).
		First(&user).Error
	if err != nil {
		return nil, err
	}
	return &user, nil
}

// UpdateFields updates selected columns for a user by ID
func (r *UserRepository) UpdateFields(ctx context.Context, id uint, fields map[string]interface{}) error {
	if len(fields) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Updates(fields).Error
}

// UpdateStatus updates only the status field for a user
func (r *UserRepository) UpdateStatus(ctx context.Context, id uint, status int) error {
	return r.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Update("status", status).Error
}

// ExportWithFilters retrieves users matching export filter criteria (with roles)
func (r *UserRepository) ExportWithFilters(ctx context.Context, filters *types.UserExportFilters) ([]models.User, error) {
	db := r.db.WithContext(ctx).Model(&models.User{})

	if filters != nil {
		if filters.Username != "" {
			db = db.Where("username LIKE ?", "%"+filters.Username+"%")
		}
		if filters.Email != "" {
			db = db.Where("email LIKE ?", "%"+filters.Email+"%")
		}
		if filters.Status != nil {
			db = db.Where("status = ?", *filters.Status)
		}
		if !filters.StartTime.IsZero() {
			db = db.Where("created_at >= ?", filters.StartTime)
		}
		if !filters.EndTime.IsZero() {
			db = db.Where("created_at <= ?", filters.EndTime)
		}
	}

	var users []models.User
	err := db.Preload("Roles").Find(&users).Error
	return users, err
}

// ErrCannotAssignAdminRole is returned when trying to assign the admin role via ReplaceUserRoles.
var ErrCannotAssignAdminRole = errors.New("cannot assign admin role through this endpoint")

// ReplaceUserRoles replaces all role assignments for a user in a transaction.
// Returns gorm.ErrRecordNotFound if the user does not exist.
// Returns ErrCannotAssignAdminRole if roleIDs contains the admin role.
func (r *UserRepository) ReplaceUserRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	return r.Transaction(ctx, func(tx *gorm.DB) error {
		var count int64
		if err := tx.Model(&models.User{}).Where("id = ?", userID).Count(&count).Error; err != nil {
			return err
		}
		if count == 0 {
			return gorm.ErrRecordNotFound
		}

		var adminRoleCount int64
		if err := tx.Model(&models.Role{}).Where("id IN ? AND code = ?", roleIDs, "admin").Count(&adminRoleCount).Error; err != nil {
			return err
		}
		if adminRoleCount > 0 {
			return ErrCannotAssignAdminRole
		}

		if err := tx.Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}

		if len(roleIDs) == 0 {
			return nil
		}

		userRoles := make([]models.UserRole, 0, len(roleIDs))
		for _, roleID := range roleIDs {
			userRoles = append(userRoles, models.UserRole{
				UserID: userID,
				RoleID: roleID,
			})
		}
		return tx.Create(&userRoles).Error
	})
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

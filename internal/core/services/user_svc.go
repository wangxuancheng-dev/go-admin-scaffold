package services

import (
	"context"
	"errors"
	"strconv"
	"time"

	"go-admin-scaffold/internal/config"
	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/types"
	"go-admin-scaffold/pkg/logger"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrUserNotFound      = errors.New("user not found")
	ErrUsernameTaken     = errors.New("username is already taken")
	ErrEmailTaken        = errors.New("email is already taken")
	ErrInvalidUserStatus = errors.New("invalid user status")
	ErrSuperAdminModify  = errors.New("super admin account cannot be modified")
	ErrSuperAdminDelete  = errors.New("super admin account cannot be deleted")
)

type LogServiceInterface interface {
	RecordOperationLog(ctx context.Context, log *models.OperationLog) error
	RecordLoginLog(ctx context.Context, userID uint, username, ip, userAgent string, status int, message string) error
	GetUserLoginHistory(ctx context.Context, userID uint, limit int) ([]models.LoginLog, error)
	GetUserOperationHistory(ctx context.Context, userID uint, limit int) ([]models.OperationLog, error)
}

type AuthServiceInterface interface {
	IsSuperAdmin(userID uint) bool
}

type UserService struct {
	userRepo UserRepository
	logSvc   LogServiceInterface
	authSvc  AuthServiceInterface
	config   *config.Config
	permInv  PermissionCacheInvalidator
}

func NewUserService(
	userRepo UserRepository,
	logSvc LogServiceInterface,
	config *config.Config,
	authSvc AuthServiceInterface,
	permInv PermissionCacheInvalidator,
) *UserService {
	return &UserService{
		userRepo: userRepo,
		logSvc:   logSvc,
		config:   config,
		authSvc:  authSvc,
		permInv:  permInv,
	}
}

func (s *UserService) invalidateUserPermissions(ctx context.Context, userID uint) {
	if s.permInv != nil {
		s.permInv.InvalidateUserPermissions(ctx, userID)
	}
}

type CreateUserRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
	Email    string `json:"email" binding:"required,email"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Status   int    `json:"status"`
	RoleIDs  []uint `json:"role_ids"`
}

type UpdateUserRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Email    string `json:"email" binding:"omitempty,email"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar"`
	Status   int    `json:"status"`
	RoleIDs  []uint `json:"role_ids"`
}

type ChangePasswordRequest struct {
	OldPassword string `json:"old_password" binding:"required"`
	NewPassword string `json:"new_password" binding:"required,min=6"`
}

// ExportUserListRequest represents the request parameters for exporting user list
type ExportUserListRequest struct {
	Username  string    `json:"username"`
	Email     string    `json:"email"`
	Status    *int      `json:"status"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
}

// IsSuperAdmin checks if a user ID is in the super admin list
func (s *UserService) IsSuperAdmin(userID uint) bool {
	if s.authSvc != nil {
		return s.authSvc.IsSuperAdmin(userID)
	}
	if s.config == nil {
		logger.Warn(context.Background(), "IsSuperAdmin: config nil", "user_id", userID)
		return false
	}
	for _, id := range s.config.SuperAdminUintIDs() {
		if id == userID {
			return true
		}
	}
	return false
}

// Create creates a new user
func (s *UserService) Create(ctx context.Context, req *CreateUserRequest) (*models.User, error) {
	// Check if username exists
	if _, err := s.userRepo.FindByUsername(ctx, req.Username); err == nil {
		return nil, ErrUsernameTaken
	}

	// Check if email exists
	if _, err := s.userRepo.FindByEmail(ctx, req.Email); err == nil {
		return nil, ErrEmailTaken
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}

	user := &models.User{
		Username: req.Username,
		Password: string(hashedPassword),
		Nickname: req.Nickname,
		Email:    req.Email,
		Avatar:   req.Avatar,
		Status:   req.Status,
	}

	// Use transaction to ensure both user creation and role assignment succeed
	err = s.userRepo.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Create user
		if err := tx.Create(user).Error; err != nil {
			return err
		}

		// If no roles specified, assign default user role
		if len(req.RoleIDs) == 0 {
			var userRole models.Role
			if err := tx.Where("code = ?", "user").First(&userRole).Error; err != nil {
				return err
			}
			req.RoleIDs = []uint{userRole.ID}
		}

		// Create user-role associations
		userRoles := make([]models.UserRole, 0, len(req.RoleIDs))
		for _, roleID := range req.RoleIDs {
			userRoles = append(userRoles, models.UserRole{
				UserID: user.ID,
				RoleID: roleID,
			})
		}
		if err := tx.Create(&userRoles).Error; err != nil {
			return err
		}

		// Load roles for the user
		return tx.Preload("Roles").First(user, user.ID).Error
	})

	if err != nil {
		return nil, err
	}

	s.invalidateUserPermissions(ctx, user.ID)

	// Record operation log
	if s.logSvc != nil {
		s.logSvc.RecordOperationLog(ctx, &models.OperationLog{
			UserID:       user.ID,
			Username:     user.Username,
			Action:       "create_user",
			Module:       "user",
			BusinessID:   strconv.FormatUint(uint64(user.ID), 10),
			BusinessType: "user",
			Status:       1,
			ErrorMessage: "",
		})
	}

	return user, nil
}

// Update updates a user
func (s *UserService) Update(ctx context.Context, id uint, req *UpdateUserRequest) (*models.User, error) {
	// First get user without roles to check basic info
	var user models.User
	if err := s.userRepo.GetDB().WithContext(ctx).Select("id", "username", "email", "nickname", "avatar", "status").Where("id = ?", id).First(&user).Error; err != nil {
		return nil, ErrUserNotFound
	}

	// Prevent modification of super admin account
	if s.IsSuperAdmin(id) {
		if req.Status != 0 || req.Username != "" || req.Password != "" || req.Avatar != "" || len(req.RoleIDs) > 0 {
			return nil, ErrSuperAdminModify
		}
		// Only allow updating nickname and email for super admin
		updateData := make(map[string]interface{})
		if req.Nickname != "" {
			updateData["nickname"] = req.Nickname
		}
		if req.Email != "" {
			if existingUser, err := s.userRepo.FindByEmail(ctx, req.Email); err == nil && existingUser.ID != id {
				return nil, ErrEmailTaken
			}
			updateData["email"] = req.Email
		}

		if len(updateData) > 0 {
			if err := s.userRepo.GetDB().WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Updates(updateData).Error; err != nil {
				return nil, err
			}
		}
	} else {
		// Normal user update logic
		updateData := make(map[string]interface{})

		if req.Email != "" && req.Email != user.Email {
			if existingUser, err := s.userRepo.FindByEmail(ctx, req.Email); err == nil && existingUser.ID != id {
				return nil, ErrEmailTaken
			}
			updateData["email"] = req.Email
		}
		if req.Nickname != "" {
			updateData["nickname"] = req.Nickname
		}
		if req.Username != "" {
			updateData["username"] = req.Username
		}
		if req.Avatar != "" {
			updateData["avatar"] = req.Avatar
		}
		if req.Status != 0 {
			updateData["status"] = req.Status
		}

		if len(updateData) > 0 {
			if err := s.userRepo.GetDB().WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Updates(updateData).Error; err != nil {
				return nil, err
			}
		}
	}

	// Record operation log
	if s.logSvc != nil {
		s.logSvc.RecordOperationLog(ctx, &models.OperationLog{
			UserID:       user.ID,
			Username:     user.Username,
			Action:       "update_user",
			Module:       "user",
			BusinessID:   strconv.FormatUint(uint64(user.ID), 10),
			BusinessType: "user",
			Status:       1,
			ErrorMessage: "",
		})
	}

	// Return updated user with roles
	return s.userRepo.FindByID(ctx, id)
}

// Delete deletes a user
func (s *UserService) Delete(ctx context.Context, id uint) error {
	// Prevent deletion of super admin account
	if s.IsSuperAdmin(id) {
		return ErrSuperAdminDelete
	}

	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return ErrUserNotFound
	}

	if err := s.userRepo.Delete(ctx, id); err != nil {
		return err
	}

	s.invalidateUserPermissions(ctx, id)

	// Record operation log
	if s.logSvc != nil {
		s.logSvc.RecordOperationLog(ctx, &models.OperationLog{
			UserID:       user.ID,
			Username:     user.Username,
			Action:       "delete_user",
			Module:       "user",
			BusinessID:   strconv.FormatUint(uint64(user.ID), 10),
			BusinessType: "user",
			Status:       1,
			ErrorMessage: "",
		})
	}

	return nil
}

// GetByID gets a user by ID
func (s *UserService) GetByID(ctx context.Context, id uint) (*models.User, error) {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return nil, ErrUserNotFound
	}
	// Set IsSuperAdmin field
	user.IsSuperAdmin = s.IsSuperAdmin(user.ID)
	return user, nil
}

// List gets a paginated list of users
func (s *UserService) List(ctx context.Context, pagination *models.Pagination) ([]models.User, error) {
	return s.userRepo.ListWithRoles(ctx, pagination)
}

// ListWithFilters gets a paginated list of users with search filters
func (s *UserService) ListWithFilters(ctx context.Context, pagination *models.Pagination, filters *types.UserSearchFilters) ([]models.User, error) {
	users, err := s.userRepo.ListWithFilters(ctx, pagination, filters)
	if err != nil {
		return nil, err
	}

	// Set IsSuperAdmin field for each user
	for i := range users {
		users[i].IsSuperAdmin = s.IsSuperAdmin(users[i].ID)
	}

	return users, nil
}

// ChangePassword changes a user's password
func (s *UserService) ChangePassword(ctx context.Context, id uint, req *ChangePasswordRequest) error {
	user, err := s.userRepo.FindByID(ctx, id)
	if err != nil {
		return ErrUserNotFound
	}

	// Verify old password
	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.OldPassword)); err != nil {
		return errors.New("old password is incorrect")
	}

	// Hash new password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.NewPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	user.Password = string(hashedPassword)
	if err := s.userRepo.Update(ctx, user); err != nil {
		return err
	}

	// Record operation log
	if s.logSvc != nil {
		s.logSvc.RecordOperationLog(ctx, &models.OperationLog{
			UserID:       user.ID,
			Username:     user.Username,
			Action:       "change_password",
			Module:       "user",
			BusinessID:   strconv.FormatUint(uint64(user.ID), 10),
			BusinessType: "user",
			Status:       1,
			ErrorMessage: "",
		})
	}

	return nil
}

// UpdateStatus updates a user's status
func (s *UserService) UpdateStatus(ctx context.Context, id uint, status int) error {
	// Prevent status modification of super admin account
	if s.IsSuperAdmin(id) {
		return ErrSuperAdminModify
	}

	// First check if user exists without loading roles
	var user models.User
	if err := s.userRepo.GetDB().WithContext(ctx).Select("id", "username", "status").Where("id = ?", id).First(&user).Error; err != nil {
		return ErrUserNotFound
	}

	if status != 0 && status != 1 {
		return ErrInvalidUserStatus
	}

	// Update only the status field to avoid affecting role associations
	if err := s.userRepo.GetDB().WithContext(ctx).Model(&models.User{}).Where("id = ?", id).Update("status", status).Error; err != nil {
		return err
	}

	// Record operation log
	if s.logSvc != nil {
		s.logSvc.RecordOperationLog(ctx, &models.OperationLog{
			UserID:       user.ID,
			Username:     user.Username,
			Action:       "update_status",
			Module:       "user",
			BusinessID:   strconv.FormatUint(uint64(user.ID), 10),
			BusinessType: "user",
			Status:       1,
			ErrorMessage: "",
		})
	}

	return nil
}

// GetUserLoginHistory gets a user's recent login history
func (s *UserService) GetUserLoginHistory(ctx context.Context, id uint, limit int) ([]models.LoginLog, error) {
	if s.logSvc == nil {
		return nil, nil
	}
	return s.logSvc.GetUserLoginHistory(ctx, id, limit)
}

// GetUserOperationHistory gets a user's recent operation history
func (s *UserService) GetUserOperationHistory(ctx context.Context, id uint, limit int) ([]models.OperationLog, error) {
	if s.logSvc == nil {
		return nil, nil
	}
	return s.logSvc.GetUserOperationHistory(ctx, id, limit)
}

// ExportUserList exports user list data based on filter criteria
func (s *UserService) ExportUserList(ctx context.Context, req *ExportUserListRequest) ([]models.User, error) {
	db := s.userRepo.GetDB().WithContext(ctx)

	// Apply filters
	if req.Username != "" {
		db = db.Where("username LIKE ?", "%"+req.Username+"%")
	}
	if req.Email != "" {
		db = db.Where("email LIKE ?", "%"+req.Email+"%")
	}
	if req.Status != nil {
		db = db.Where("status = ?", *req.Status)
	}
	if !req.StartTime.IsZero() {
		db = db.Where("created_at >= ?", req.StartTime)
	}
	if !req.EndTime.IsZero() {
		db = db.Where("created_at <= ?", req.EndTime)
	}

	var users []models.User
	err := db.Preload("Roles").Find(&users).Error
	if err != nil {
		return nil, err
	}

	// Record operation log
	if s.logSvc != nil {
		s.logSvc.RecordOperationLog(ctx, &models.OperationLog{
			Action:       "export_users",
			Module:       "user",
			BusinessType: "user",
			Status:       1,
			ErrorMessage: "",
		})
	}

	return users, nil
}

// UpdateUserRoles updates a user's role assignments
func (s *UserService) UpdateUserRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	// Prevent role modification of super admin account
	if s.IsSuperAdmin(userID) {
		return ErrSuperAdminModify
	}

	err := s.userRepo.GetDB().WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		// Check if user exists
		user, err := s.userRepo.FindByID(ctx, userID)
		if err != nil {
			return ErrUserNotFound
		}

		// Check if any of the roles is admin role
		var adminRoleCount int64
		if err := tx.Model(&models.Role{}).Where("id IN ? AND code = ?", roleIDs, "admin").Count(&adminRoleCount).Error; err != nil {
			return err
		}

		// Prevent assigning admin role through this endpoint
		if adminRoleCount > 0 {
			return errors.New("cannot assign admin role through this endpoint")
		}

		// Remove existing role assignments
		if err := tx.Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil {
			return err
		}

		// Add new role assignments
		if len(roleIDs) > 0 {
			userRoles := make([]models.UserRole, 0, len(roleIDs))
			for _, roleID := range roleIDs {
				userRoles = append(userRoles, models.UserRole{
					UserID: userID,
					RoleID: roleID,
				})
			}
			if err := tx.Create(&userRoles).Error; err != nil {
				return err
			}
		}

		// Record operation log
		if s.logSvc != nil {
			s.logSvc.RecordOperationLog(ctx, &models.OperationLog{
				UserID:       user.ID,
				Username:     user.Username,
				Action:       "update_user_roles",
				Module:       "user",
				BusinessID:   strconv.FormatUint(uint64(user.ID), 10),
				BusinessType: "user",
				Status:       1,
				ErrorMessage: "",
			})
		}

		return nil
	})
	if err != nil {
		return err
	}
	s.invalidateUserPermissions(ctx, userID)
	return nil
}

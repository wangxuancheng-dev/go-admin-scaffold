package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"app/internal/core/models"
	"app/internal/core/services"
	"app/internal/core/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockUserService is a mock implementation of services.UserServiceAPI
type MockUserService struct {
	mock.Mock
}

func (m *MockUserService) ListWithFilters(ctx context.Context, pagination *models.Pagination, filters *types.UserSearchFilters) ([]models.User, error) {
	args := m.Called(ctx, pagination, filters)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserService) Create(ctx context.Context, req *services.CreateUserRequest) (*models.User, error) {
	args := m.Called(ctx, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) Update(ctx context.Context, id uint, req *services.UpdateUserRequest) (*models.User, error) {
	args := m.Called(ctx, id, req)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) Delete(ctx context.Context, id uint) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockUserService) GetByID(ctx context.Context, id uint) (*models.User, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserService) ExportUserList(ctx context.Context, req *services.ExportUserListRequest) ([]models.User, error) {
	args := m.Called(ctx, req)
	return args.Get(0).([]models.User), args.Error(1)
}

func (m *MockUserService) UpdateUserRoles(ctx context.Context, userID uint, roleIDs []uint) error {
	args := m.Called(ctx, userID, roleIDs)
	return args.Error(0)
}

func (m *MockUserService) UpdateStatus(ctx context.Context, id uint, status int) error {
	args := m.Called(ctx, id, status)
	return args.Error(0)
}

func (m *MockUserService) IsSuperAdmin(userID uint) bool {
	args := m.Called(userID)
	return args.Bool(0)
}

func setupTestRouter() (*gin.Engine, *MockUserService) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mockSvc := new(MockUserService)
	r.Use(func(c *gin.Context) {
		c.Set("userService", mockSvc)
	})
	return r, mockSvc
}

func TestListUsers(t *testing.T) {
	r, mockSvc := setupTestRouter()
	r.GET("/users", ListUsers)

	users := []models.User{
		{ID: 1, Username: "user1", Email: "user1@example.com"},
		{ID: 2, Username: "user2", Email: "user2@example.com"},
	}

	mockSvc.On("ListWithFilters", mock.Anything, mock.AnythingOfType("*models.Pagination"), mock.AnythingOfType("*types.UserSearchFilters")).
		Return(users, nil)
	mockSvc.On("IsSuperAdmin", uint(1)).Return(false)
	mockSvc.On("IsSuperAdmin", uint(2)).Return(false)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/users?page=1&page_size=10", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotNil(t, data["items"])
	assert.NotNil(t, data["pagination"])
}

func TestExportUsers(t *testing.T) {
	r, mockSvc := setupTestRouter()
	r.POST("/users/export", ExportUsers)

	now := time.Now()
	req := services.ExportUserListRequest{
		Username:  "user",
		Email:     "@example.com",
		Status:    &[]int{1}[0],
		StartTime: now.Add(-24 * time.Hour),
		EndTime:   now,
	}

	exported := []models.User{
		{ID: 1, Username: "user1", Email: "user1@example.com", Status: 1, Roles: []models.Role{{Name: "admin"}}},
		{ID: 2, Username: "user2", Email: "user2@example.com", Status: 1, Roles: []models.Role{{Name: "user"}}},
	}

	mockSvc.On("ExportUserList", mock.Anything, mock.AnythingOfType("*services.ExportUserListRequest")).
		Return(exported, nil)

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/users/export", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &response)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), response["code"])
	assert.Equal(t, "success", response["message"])

	data, ok := response["data"].([]interface{})
	assert.True(t, ok)
	assert.Len(t, data, 2)
}

func TestCreateUser(t *testing.T) {
	r, mockSvc := setupTestRouter()
	r.POST("/users", CreateUser)

	req := services.CreateUserRequest{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
		Nickname: "Test User",
		Status:   1,
	}

	createdUser := &models.User{
		ID:       1,
		Username: req.Username,
		Email:    req.Email,
		Nickname: req.Nickname,
		Status:   req.Status,
	}

	mockSvc.On("Create", mock.Anything, mock.AnythingOfType("*services.CreateUserRequest")).
		Return(createdUser, nil)

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/users", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var envelope struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &envelope)
	assert.NoError(t, err)
	assert.Equal(t, 0, envelope.Code)
	assert.Equal(t, createdUser.Username, envelope.Data["username"])
	assert.Equal(t, createdUser.Email, envelope.Data["email"])
}

func TestUpdateUser(t *testing.T) {
	r, mockSvc := setupTestRouter()
	r.PUT("/users/:id", UpdateUser)

	req := services.UpdateUserRequest{
		Nickname: "Updated User",
		Email:    "updated@example.com",
		Status:   1,
	}

	updatedUser := &models.User{
		ID:       1,
		Username: "testuser",
		Email:    req.Email,
		Nickname: req.Nickname,
		Status:   req.Status,
	}

	mockSvc.On("Update", mock.Anything, uint(1), mock.AnythingOfType("*services.UpdateUserRequest")).
		Return(updatedUser, nil)

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("PUT", "/users/1", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)

	var envelope struct {
		Code int                    `json:"code"`
		Data map[string]interface{} `json:"data"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &envelope)
	assert.NoError(t, err)
	assert.Equal(t, 0, envelope.Code)
	assert.Equal(t, updatedUser.Nickname, envelope.Data["nickname"])
	assert.Equal(t, updatedUser.Email, envelope.Data["email"])
}

func TestDeleteUser(t *testing.T) {
	r, mockSvc := setupTestRouter()
	r.DELETE("/users/:id", DeleteUser)

	mockSvc.On("Delete", mock.Anything, uint(1)).Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/users/1", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var envelope struct {
		Code int `json:"code"`
	}
	err := json.Unmarshal(w.Body.Bytes(), &envelope)
	assert.NoError(t, err)
	assert.Equal(t, 0, envelope.Code)
}

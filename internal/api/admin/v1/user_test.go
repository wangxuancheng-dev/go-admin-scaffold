package v1

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"go-admin-scaffold/internal/core/models"
	"go-admin-scaffold/internal/core/services"
	"go-admin-scaffold/internal/core/types"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

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
	return m.Called(ctx, id).Error(0)
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
	return m.Called(ctx, userID, roleIDs).Error(0)
}

func (m *MockUserService) UpdateStatus(ctx context.Context, id uint, status int) error {
	return m.Called(ctx, id, status).Error(0)
}

func (m *MockUserService) IsSuperAdmin(userID uint) bool {
	return m.Called(userID).Bool(0)
}

func setupUserHandler() (*gin.Engine, *MockUserService, *UserHandler) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	mockSvc := new(MockUserService)
	h := NewUserHandler(mockSvc, nil)
	return r, mockSvc, h
}

func TestListUsers(t *testing.T) {
	r, mockSvc, h := setupUserHandler()
	r.GET("/users", h.ListUsers)

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
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	data, ok := response["data"].(map[string]interface{})
	assert.True(t, ok)
	assert.NotNil(t, data["items"])
	assert.NotNil(t, data["pagination"])
}

func TestExportUsers(t *testing.T) {
	r, mockSvc, h := setupUserHandler()
	r.POST("/users/export", h.ExportUsers)

	now := time.Now()
	reqBody := services.ExportUserListRequest{
		Username:  "user",
		Email:     "@example.com",
		Status:    &[]int{1}[0],
		StartTime: now.Add(-24 * time.Hour),
		EndTime:   now,
	}
	exported := []models.User{
		{ID: 1, Username: "user1", Email: "user1@example.com", Status: 1},
		{ID: 2, Username: "user2", Email: "user2@example.com", Status: 1},
	}
	mockSvc.On("ExportUserList", mock.Anything, mock.AnythingOfType("*services.ExportUserListRequest")).
		Return(exported, nil)

	body, _ := json.Marshal(reqBody)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/users/export", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
	var response map[string]interface{}
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &response))
	assert.Equal(t, float64(0), response["code"])
}

func TestCreateUser(t *testing.T) {
	r, mockSvc, h := setupUserHandler()
	r.POST("/users", h.CreateUser)

	req := services.CreateUserRequest{
		Username: "testuser",
		Password: "password123",
		Email:    "test@example.com",
		Nickname: "Test User",
		Status:   1,
	}
	createdUser := &models.User{ID: 1, Username: req.Username, Email: req.Email, Nickname: req.Nickname, Status: req.Status}
	mockSvc.On("Create", mock.Anything, mock.AnythingOfType("*services.CreateUserRequest")).Return(createdUser, nil)

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
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &envelope))
	assert.Equal(t, 0, envelope.Code)
	assert.Equal(t, createdUser.Username, envelope.Data["username"])
}

func TestUpdateUser(t *testing.T) {
	r, mockSvc, h := setupUserHandler()
	r.PUT("/users/:id", h.UpdateUser)

	req := services.UpdateUserRequest{Nickname: "Updated User", Email: "updated@example.com", Status: 1}
	updatedUser := &models.User{ID: 1, Username: "testuser", Email: req.Email, Nickname: req.Nickname, Status: req.Status}
	mockSvc.On("Update", mock.Anything, uint(1), mock.AnythingOfType("*services.UpdateUserRequest")).Return(updatedUser, nil)

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("PUT", "/users/1", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDeleteUser(t *testing.T) {
	r, mockSvc, h := setupUserHandler()
	r.DELETE("/users/:id", h.DeleteUser)
	mockSvc.On("Delete", mock.Anything, uint(1)).Return(nil)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/users/1", nil)
	r.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)
}

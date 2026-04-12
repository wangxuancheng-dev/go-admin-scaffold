package services

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"app/internal/config"
	"app/internal/core/models"
	"app/pkg/logger"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUserInactive       = errors.New("user is inactive")
)

type AuthService struct {
	userRepo UserRepository
	logSvc   *LogService
	config   *config.Config
}

func NewAuthService(userRepo UserRepository, logSvc *LogService, config *config.Config) *AuthService {
	return &AuthService{
		userRepo: userRepo,
		logSvc:   logSvc,
		config:   config,
	}
}

// IsSuperAdmin checks if a user ID is in the super admin list (parsed at config load).
func (s *AuthService) IsSuperAdmin(userID uint) bool {
	if s.config == nil {
		return false
	}
	for _, id := range s.config.SuperAdminUintIDs() {
		if id == userID {
			return true
		}
	}
	return false
}

type LoginRequest struct {
	Username    string `json:"username" binding:"required"`
	Password    string `json:"password" binding:"required"`
	CaptchaID   string `json:"captcha_id" binding:"required"`
	CaptchaCode string `json:"captcha_code" binding:"required"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int    `json:"expires_in"`
}

// ValidateToken validates a JWT and returns claims.
func (s *AuthService) ValidateToken(tokenString string) (jwt.MapClaims, error) {
	claims := jwt.MapClaims{}
	token, err := jwt.ParseWithClaims(tokenString, &claims, func(token *jwt.Token) (interface{}, error) {
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return []byte(s.config.JWT.Secret), nil
	}, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	if err != nil {
		return nil, err
	}
	if !token.Valid {
		return nil, errors.New("invalid token claims")
	}
	return claims, nil
}

// GetUserFromClaims loads the user from JWT claims.
func (s *AuthService) GetUserFromClaims(ctx context.Context, claims jwt.MapClaims) (*models.User, error) {
	userIDValue, exists := claims["user_id"]
	if !exists {
		logger.Warn(ctx, "jwt missing user_id claim")
		return nil, errors.New("user_id not found in claims")
	}

	var userID uint
	switch v := userIDValue.(type) {
	case float64:
		userID = uint(v)
	case float32:
		userID = uint(v)
	case int:
		userID = uint(v)
	case int64:
		userID = uint(v)
	case uint:
		userID = v
	case uint64:
		userID = uint(v)
	default:
		logger.Warn(ctx, "jwt invalid user_id type", "type", fmt.Sprintf("%T", userIDValue))
		return nil, errors.New("invalid user_id type in claims")
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		logger.Error(ctx, "jwt user lookup failed", "error", err, "user_id", userID)
		return nil, err
	}

	user.IsSuperAdmin = s.IsSuperAdmin(user.ID)
	return user, nil
}

func (s *AuthService) Login(ctx context.Context, req *LoginRequest) (*TokenResponse, error) {
	user, err := s.userRepo.FindByUsername(ctx, req.Username)
	if err != nil {
		if s.logSvc != nil {
			s.logSvc.RecordLoginLog(ctx, 0, req.Username, "", "", 0, "user not found")
		}
		return nil, ErrInvalidCredentials
	}

	if user.Status == 0 {
		if s.logSvc != nil {
			s.logSvc.RecordLoginLog(ctx, user.ID, user.Username, "", "", 0, "user is inactive")
		}
		return nil, ErrUserInactive
	}

	if !s.validatePassword(user.Password, req.Password) {
		if s.logSvc != nil {
			s.logSvc.RecordLoginLog(ctx, user.ID, user.Username, "", "", 0, "invalid password")
		}
		return nil, ErrInvalidCredentials
	}

	user.IsSuperAdmin = s.IsSuperAdmin(user.ID)

	if err := s.userRepo.UpdateLastLogin(ctx, user.ID); err != nil {
		logger.Warn(ctx, "update last login failed", "error", err, "user_id", user.ID)
	}

	token, err := s.generateToken(user)
	if err != nil {
		return nil, err
	}

	if s.logSvc != nil {
		s.logSvc.RecordLoginLog(ctx, user.ID, user.Username, "", "", 1, "login successful")
	}

	return &TokenResponse{
		AccessToken: token,
		TokenType:   "Bearer",
		ExpiresIn:   s.config.JWT.ExpireTime,
	}, nil
}

func (s *AuthService) validatePassword(hashedPassword, plainPassword string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(plainPassword))
	return err == nil
}

func (s *AuthService) generateToken(user *models.User) (string, error) {
	now := time.Now()
	claims := jwt.MapClaims{
		"user_id":  user.ID,
		"username": user.Username,
		"iat":      now.Unix(),
		"exp":      now.Add(time.Second * time.Duration(s.config.JWT.ExpireTime)).Unix(),
	}
	if iss := strings.TrimSpace(s.config.JWT.Issuer); iss != "" {
		claims["iss"] = iss
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(s.config.JWT.Secret))
}

func (s *AuthService) HashPassword(password string) (string, error) {
	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}
	return string(hashedBytes), nil
}

// RefreshToken issues a new access token for the user.
func (s *AuthService) RefreshToken(ctx context.Context, userID uint) (string, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return "", err
	}

	if user.Status != 1 {
		return "", ErrUserInactive
	}

	user.IsSuperAdmin = s.IsSuperAdmin(user.ID)
	return s.generateToken(user)
}

// GetConfig returns application config (avoid exposing in new code; kept for handlers).
func (s *AuthService) GetConfig() *config.Config {
	return s.config
}

// Logout records logout.
func (s *AuthService) Logout(ctx context.Context, userID uint) error {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return err
	}

	if s.logSvc != nil {
		return s.logSvc.RecordLoginLog(ctx, user.ID, user.Username, "", "", 1, "logout successful")
	}

	return nil
}

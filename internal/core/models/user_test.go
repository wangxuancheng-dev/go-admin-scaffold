package models_test

import (
	"testing"

	"go-admin-scaffold/internal/core/models"

	"github.com/stretchr/testify/require"
	"golang.org/x/crypto/bcrypt"
)

func TestUser_ValidatePassword(t *testing.T) {
	hash, err := bcrypt.GenerateFromPassword([]byte("secret"), bcrypt.MinCost)
	require.NoError(t, err)

	u := &models.User{Password: string(hash)}
	require.True(t, u.ValidatePassword("secret"))
	require.False(t, u.ValidatePassword("wrong"))
	require.False(t, (*models.User)(nil).ValidatePassword("secret"))
	require.False(t, (&models.User{}).ValidatePassword("secret"))
}

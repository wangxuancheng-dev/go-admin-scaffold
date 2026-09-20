package database_test

import (
	"context"
	"testing"

	"go-admin-scaffold/pkg/database"

	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestWithContext_FromContext(t *testing.T) {
	db := &gorm.DB{}
	ctx := database.WithContext(context.Background(), db)
	got, err := database.FromContext(ctx)
	require.NoError(t, err)
	require.Equal(t, db, got)
}

func TestFromContext_missing(t *testing.T) {
	_, err := database.FromContext(context.Background())
	require.Error(t, err)
}

func TestMustFromContext(t *testing.T) {
	db := &gorm.DB{}
	ctx := database.WithContext(context.Background(), db)
	require.Equal(t, db, database.MustFromContext(ctx))
}

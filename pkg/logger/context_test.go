package logger_test

import (
	"context"
	"testing"

	"go-admin-scaffold/pkg/logger"

	"github.com/stretchr/testify/require"
)

func TestContextLogger_DefaultAndNop(t *testing.T) {
	ctx := context.Background()
	require.NotNil(t, logger.Default())
	logger.Default().Debug(ctx, "d")
	logger.Default().Info(ctx, "i")
	logger.Default().Warn(ctx, "w")
	logger.Default().Error(ctx, "e")

	nop := logger.Nop()
	require.NotNil(t, nop)
	nop.Debug(ctx, "d")
	nop.Info(ctx, "i")
	nop.Warn(ctx, "w")
	nop.Error(ctx, "e")
}

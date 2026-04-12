package logger

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	lumberjackV2 "gopkg.in/natefinch/lumberjack.v2"
)

type contextKey string

const traceIDKey contextKey = "trace_id"

type Config struct {
	Level      string `yaml:"level"`       // 日志级别
	Filename   string `yaml:"filename"`    // 日志文件路径
	MaxSize    int    `yaml:"max_size"`    // 每个日志文件最大尺寸，单位MB
	MaxBackups int    `yaml:"max_backups"` // 保留的旧日志文件最大数量
	MaxAge     int    `yaml:"max_age"`     // 保留的旧日志文件最大天数
	Compress   bool   `yaml:"compress"`    // 是否压缩旧日志文件
	Daily      bool   `yaml:"daily"`       // 是否按天切割日志
	// Timezone IANA name for daily file date and max_age cleanup (e.g. Asia/Shanghai, UTC). Empty = time.Local.
	Timezone string `yaml:"timezone"`
}

var (
	logger *zap.Logger
	sugar  *zap.SugaredLogger

	// currentDailyRotator is set when Setup uses daily rotation; closed in Close().
	currentDailyRotator *dailyRotateWriter
)

func init() {
	logger = zap.NewNop()
	sugar = logger.Sugar()
}

// Sugared returns the process-wide sugared logger (no-op until Setup completes).
func Sugared() *zap.SugaredLogger {
	return sugar
}

// Setup initializes the logger
func Setup(config *Config) error {
	currentDailyRotator = nil

	if tz := strings.TrimSpace(config.Timezone); tz != "" && !strings.EqualFold(tz, "local") {
		if _, err := time.LoadLocation(tz); err != nil {
			return fmt.Errorf("log.timezone %q: %w", tz, err)
		}
	}

	// Create encoder config
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	writer, err := openLogWriter(config)
	if err != nil {
		return err
	}
	currentDailyRotator = nil
	if config.Daily {
		if dw, ok := writer.(*dailyRotateWriter); ok {
			currentDailyRotator = dw
		}
	}

	// Parse log level
	level, parseErr := zapcore.ParseLevel(config.Level)
	if parseErr != nil {
		level = zapcore.InfoLevel
	}

	// Create core
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writer,
		level,
	)

	// Create logger
	logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
	sugar = logger.Sugar()

	return nil
}

// openLogWriter creates the file sink for Setup / LogBuilder (MkdirAll errors are returned, not panicked).
func openLogWriter(config *Config) (zapcore.WriteSyncer, error) {
	logDir := filepath.Dir(config.Filename)
	if err := os.MkdirAll(logDir, 0750); err != nil {
		return nil, fmt.Errorf("create log directory: %w", err)
	}
	if config.Daily {
		return newDailyRotateWriter(config), nil
	}
	return zapcore.AddSync(&lumberjackV2.Logger{
		Filename:   config.Filename,
		MaxSize:    config.MaxSize,
		MaxBackups: config.MaxBackups,
		MaxAge:     config.MaxAge,
		Compress:   config.Compress,
		LocalTime:  true,
	}), nil
}

// WithField adds a field to the logger context
func WithField(ctx context.Context, key string, value interface{}) context.Context {
	return context.WithValue(ctx, contextKey(key), value)
}

// getTraceID gets trace ID from context
func getTraceID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	if traceID, ok := ctx.Value(traceIDKey).(string); ok {
		return traceID
	}
	return ""
}

// Debug logs a debug message
func Debug(ctx context.Context, msg string, args ...interface{}) {
	sugar.Debugw(msg, append(args, "trace_id", getTraceID(ctx))...)
}

// Info logs an info message
func Info(ctx context.Context, msg string, args ...interface{}) {
	sugar.Infow(msg, append(args, "trace_id", getTraceID(ctx))...)
}

// Warn logs a warning message
func Warn(ctx context.Context, msg string, args ...interface{}) {
	sugar.Warnw(msg, append(args, "trace_id", getTraceID(ctx))...)
}

// Error logs an error message
func Error(ctx context.Context, msg string, args ...interface{}) {
	sugar.Errorw(msg, append(args, "trace_id", getTraceID(ctx))...)
}

// Fatal logs a fatal message and exits
func Fatal(ctx context.Context, msg string, args ...interface{}) {
	sugar.Fatalw(msg, append(args, "trace_id", getTraceID(ctx))...)
}

// Close flushes any buffered log entries
func Close() error {
	if currentDailyRotator != nil {
		_ = currentDailyRotator.Close()
		currentDailyRotator = nil
	}
	if logger == nil {
		return nil
	}
	return logger.Sync()
}

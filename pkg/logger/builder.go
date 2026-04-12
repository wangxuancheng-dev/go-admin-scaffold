package logger

import (
	"fmt"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// LogBuilder 用于构建日志实例
type LogBuilder struct {
	config *Config
}

// NewBuilder 创建一个新的日志构建器
func NewBuilder() *LogBuilder {
	return &LogBuilder{
		config: &Config{
			Level:      "info",
			MaxSize:    100,
			MaxBackups: 10,
			MaxAge:     30,
			Compress:   true,
			Daily:      true,
		},
	}
}

// SetDriver 设置日志驱动类型
func (b *LogBuilder) SetDriver(driver string) *LogBuilder {
	if driver == "daily" {
		b.config.Daily = true
	}
	return b
}

// SetPath 设置日志文件路径
func (b *LogBuilder) SetPath(path string) *LogBuilder {
	b.config.Filename = path
	return b
}

// SetLevel 设置日志级别
func (b *LogBuilder) SetLevel(level string) *LogBuilder {
	b.config.Level = level
	return b
}

// SetMaxSize 设置单个日志文件最大尺寸(MB)
func (b *LogBuilder) SetMaxSize(size int) *LogBuilder {
	b.config.MaxSize = size
	return b
}

// SetMaxBackups 设置最大备份数
func (b *LogBuilder) SetMaxBackups(count int) *LogBuilder {
	b.config.MaxBackups = count
	return b
}

// SetMaxAge 设置日志保留天数
func (b *LogBuilder) SetMaxAge(days int) *LogBuilder {
	b.config.MaxAge = days
	return b
}

// SetCompress 设置是否压缩
func (b *LogBuilder) SetCompress(compress bool) *LogBuilder {
	b.config.Compress = compress
	return b
}

// Build 构建并返回日志实例
func (b *LogBuilder) Build() (*Logger, error) {
	writer, err := openLogWriter(b.config)
	if err != nil {
		return nil, fmt.Errorf("log writer: %w", err)
	}

	// 设置日志编码配置
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

	// 设置日志级别
	var level zapcore.Level
	switch b.config.Level {
	case "debug":
		level = zapcore.DebugLevel
	case "info":
		level = zapcore.InfoLevel
	case "warn":
		level = zapcore.WarnLevel
	case "error":
		level = zapcore.ErrorLevel
	default:
		level = zapcore.InfoLevel
	}

	// 创建 Core
	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		writer,
		level,
	)

	// 创建 Logger
	logger := &Logger{
		logger: zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1)),
	}

	return logger, nil
}

package logger

import "context"

// ContextLogger is the injectable logging API used on HTTP and service paths.
// Package-level Debug/Info/Warn/Error remain for bootstrap and CLI; prefer injecting
// ContextLogger at composition roots so tests can substitute a no-op or spy.
type ContextLogger interface {
	Debug(ctx context.Context, msg string, args ...interface{})
	Info(ctx context.Context, msg string, args ...interface{})
	Warn(ctx context.Context, msg string, args ...interface{})
	Error(ctx context.Context, msg string, args ...interface{})
}

type stdContextLogger struct{}

func (stdContextLogger) Debug(ctx context.Context, msg string, args ...interface{}) {
	Debug(ctx, msg, args...)
}
func (stdContextLogger) Info(ctx context.Context, msg string, args ...interface{}) {
	Info(ctx, msg, args...)
}
func (stdContextLogger) Warn(ctx context.Context, msg string, args ...interface{}) {
	Warn(ctx, msg, args...)
}
func (stdContextLogger) Error(ctx context.Context, msg string, args ...interface{}) {
	Error(ctx, msg, args...)
}

// Default returns the process-wide ContextLogger backed by Setup().
func Default() ContextLogger {
	return stdContextLogger{}
}

// Nop returns a ContextLogger that discards all messages (tests).
func Nop() ContextLogger {
	return nopContextLogger{}
}

type nopContextLogger struct{}

func (nopContextLogger) Debug(context.Context, string, ...interface{}) {}
func (nopContextLogger) Info(context.Context, string, ...interface{})  {}
func (nopContextLogger) Warn(context.Context, string, ...interface{})  {}
func (nopContextLogger) Error(context.Context, string, ...interface{}) {}

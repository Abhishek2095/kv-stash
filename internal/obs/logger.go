// Package obs provides observability components including logging and metrics for the kv-stash server.
package obs

import (
	"log/slog"
	"os"
)

// Logger wraps slog.Logger for structured logging
type Logger struct {
	*slog.Logger
}

// NewLogger creates a new structured logger
func NewLogger(debug bool) *Logger {
	level := slog.LevelInfo
	if debug {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		Level: level,
	})

	return &Logger{
		Logger: slog.New(handler),
	}
}

// WithFields adds fields to the logger context
func (l *Logger) WithFields(args ...any) *Logger {
	return &Logger{
		Logger: l.With(args...),
	}
}

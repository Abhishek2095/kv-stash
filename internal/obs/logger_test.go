package obs_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/Abhishek2095/kv-stash/internal/obs"
)

func TestNewLogger(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		debug bool
	}{
		{
			name:  "Debug logger",
			debug: true,
		},
		{
			name:  "Info logger",
			debug: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			logger := obs.NewLogger(tt.debug)
			if logger == nil {
				t.Error("NewLogger returned nil")
				return
			}

			// Test that logger can be used without panic
			logger.Info("test message")
			logger.Error("test error")

			if tt.debug {
				logger.Debug("test debug message")
			}
		})
	}
}

func TestLogger_WithFields(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)

	// Test WithFields doesn't panic and returns a logger
	enrichedLogger := logger.WithFields("key", "value")
	if enrichedLogger == nil {
		t.Error("WithFields returned nil")
		return
	}

	// Test multiple fields
	multiFieldLogger := logger.WithFields("key1", "value1", "key2", "value2")
	if multiFieldLogger == nil {
		t.Error("WithFields with multiple fields returned nil")
		return
	}

	// Test that the enriched logger can be used
	enrichedLogger.Info("test message with fields")
	multiFieldLogger.Error("test error with multiple fields")
}

func TestLogger_LogLevels(t *testing.T) {
	t.Parallel()

	var buf bytes.Buffer

	// Create a logger that writes to our buffer
	handler := slog.NewTextHandler(&buf, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	})

	logger := &obs.Logger{
		Logger: slog.New(handler),
	}

	// Test different log levels
	logger.Debug("debug message")
	logger.Info("info message")
	logger.Warn("warn message")
	logger.Error("error message")

	output := buf.String()

	// Verify all messages were logged
	expectedMessages := []string{
		"debug message",
		"info message",
		"warn message",
		"error message",
	}

	for _, msg := range expectedMessages {
		if !strings.Contains(output, msg) {
			t.Errorf("Expected log output to contain %q, but it didn't. Output: %s", msg, output)
		}
	}
}

func TestLogger_WithFieldsChaining(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)

	// Test chaining WithFields calls
	chainedLogger := logger.WithFields("component", "test").WithFields("operation", "chaining")

	if chainedLogger == nil {
		t.Error("Chained WithFields returned nil")
		return
	}

	// Test that chained logger works
	chainedLogger.Info("chained logger test")
}

func TestLogger_FieldTypes(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)

	// Test various field types
	testCases := []struct {
		name   string
		fields []any
	}{
		{
			name:   "string fields",
			fields: []any{"key", "value"},
		},
		{
			name:   "int fields",
			fields: []any{"count", 42},
		},
		{
			name:   "bool fields",
			fields: []any{"enabled", true},
		},
		{
			name:   "mixed fields",
			fields: []any{"string", "value", "int", 123, "bool", false},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			enrichedLogger := logger.WithFields(tc.fields...)
			if enrichedLogger == nil {
				t.Errorf("WithFields returned nil for %s", tc.name)
				return
			}

			// Test that logger works with these field types
			enrichedLogger.Info("test message with various field types")
		})
	}
}

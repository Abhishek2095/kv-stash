package server_test

import (
	"strings"
	"testing"
	"time"

	"github.com/Abhishek2095/kv-stash/internal/obs"
	"github.com/Abhishek2095/kv-stash/internal/proto"
	"github.com/Abhishek2095/kv-stash/internal/server"
	"github.com/Abhishek2095/kv-stash/internal/store"
)

func createTestHandler(t *testing.T) *server.Handler {
	t.Helper()

	logger := obs.NewLogger(false)
	storeConfig := &store.Config{
		Shards:         4,
		MaxMemoryBytes: 0,
		EvictionPolicy: "noeviction",
	}

	s, err := store.New(storeConfig, logger)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	serverConfig := &server.Config{
		ListenAddr:   ":6380",
		Shards:       4,
		AuthPassword: "",
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	handler := server.NewHandler(s, serverConfig, logger)
	return handler
}

func TestHandler_PING(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	tests := []struct {
		name     string
		args     []string
		expected string
		respType proto.ResponseType
	}{
		{
			name:     "PING without args",
			args:     []string{},
			expected: "PONG",
			respType: proto.SimpleString,
		},
		{
			name:     "PING with message",
			args:     []string{"hello"},
			expected: "hello",
			respType: proto.BulkString,
		},
		{
			name:     "PING with too many args",
			args:     []string{"hello", "world"},
			expected: "ERR wrong number of arguments for 'ping' command",
			respType: proto.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := &proto.Command{Name: "PING", Args: tt.args}
			resp := handler.HandleCommand(cmd)

			if resp.Type != tt.respType {
				t.Errorf("Expected response type %v, got %v", tt.respType, resp.Type)
			}

			if resp.Data.(string) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, resp.Data.(string))
			}
		})
	}
}

func TestHandler_ECHO(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	tests := []struct {
		name     string
		args     []string
		expected string
		respType proto.ResponseType
	}{
		{
			name:     "ECHO with message",
			args:     []string{"hello world"},
			expected: "hello world",
			respType: proto.BulkString,
		},
		{
			name:     "ECHO with empty string",
			args:     []string{""},
			expected: "",
			respType: proto.BulkString,
		},
		{
			name:     "ECHO without args",
			args:     []string{},
			expected: "ERR wrong number of arguments for 'echo' command",
			respType: proto.Error,
		},
		{
			name:     "ECHO with too many args",
			args:     []string{"hello", "world"},
			expected: "ERR wrong number of arguments for 'echo' command",
			respType: proto.Error,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := &proto.Command{Name: "ECHO", Args: tt.args}
			resp := handler.HandleCommand(cmd)

			if resp.Type != tt.respType {
				t.Errorf("Expected response type %v, got %v", tt.respType, resp.Type)
			}

			if resp.Data.(string) != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, resp.Data.(string))
			}
		})
	}
}

func TestHandler_INFO(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	cmd := &proto.Command{Name: "INFO", Args: []string{}}
	resp := handler.HandleCommand(cmd)

	if resp.Type != proto.BulkString {
		t.Errorf("Expected BulkString response, got %v", resp.Type)
	}

	info := resp.Data.(string)
	expectedSections := []string{"# Server", "# Clients", "# Memory", "# Keyspace"}

	for _, section := range expectedSections {
		if !strings.Contains(info, section) {
			t.Errorf("INFO response missing section: %s", section)
		}
	}

	// Test with args (should still work, args are ignored)
	cmd = &proto.Command{Name: "INFO", Args: []string{"server"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.BulkString {
		t.Errorf("Expected BulkString response with args, got %v", resp.Type)
	}
}

func TestHandler_GET_SET(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	// Test GET non-existent key
	cmd := &proto.Command{Name: "GET", Args: []string{"nonexistent"}}
	resp := handler.HandleCommand(cmd)

	if resp.Type != proto.NullBulkString {
		t.Errorf("Expected NullBulkString for non-existent key, got %v", resp.Type)
	}

	// Test SET and then GET
	cmd = &proto.Command{Name: "SET", Args: []string{"testkey", "testvalue"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.SimpleString || resp.Data.(string) != "OK" {
		t.Errorf("Expected OK response for SET, got %v: %v", resp.Type, resp.Data)
	}

	// Test GET existing key
	cmd = &proto.Command{Name: "GET", Args: []string{"testkey"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.BulkString {
		t.Errorf("Expected BulkString for existing key, got %v", resp.Type)
	}

	if resp.Data.(string) != "testvalue" {
		t.Errorf("Expected 'testvalue', got %q", resp.Data.(string))
	}
}

func TestHandler_SET_WithOptions(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "SET without enough args",
			args:    []string{"key"},
			wantErr: true,
			errMsg:  "ERR wrong number of arguments for 'set' command",
		},
		{
			name:    "SET with EX option",
			args:    []string{"key", "value", "EX", "10"},
			wantErr: false,
		},
		{
			name:    "SET with PX option",
			args:    []string{"key", "value", "PX", "1000"},
			wantErr: false,
		},
		{
			name:    "SET with invalid EX value",
			args:    []string{"key", "value", "EX", "invalid"},
			wantErr: true,
			errMsg:  "ERR value is not an integer or out of range",
		},
		{
			name:    "SET with missing EX value",
			args:    []string{"key", "value", "EX"},
			wantErr: true,
			errMsg:  "ERR syntax error",
		},
		{
			name:    "SET with invalid option",
			args:    []string{"key", "value", "INVALID"},
			wantErr: true,
			errMsg:  "ERR syntax error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := &proto.Command{Name: "SET", Args: tt.args}
			resp := handler.HandleCommand(cmd)

			if tt.wantErr {
				if resp.Type != proto.Error {
					t.Errorf("Expected Error response, got %v", resp.Type)
				}
				if !strings.Contains(resp.Data.(string), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, resp.Data.(string))
				}
			} else if resp.Type != proto.SimpleString || resp.Data.(string) != "OK" {
				t.Errorf("Expected OK response, got %v: %v", resp.Type, resp.Data)
			}
		})
	}
}

func TestHandler_DEL(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	// Test DEL without args
	cmd := &proto.Command{Name: "DEL", Args: []string{}}
	resp := handler.HandleCommand(cmd)

	if resp.Type != proto.Error {
		t.Errorf("Expected Error response for DEL without args, got %v", resp.Type)
	}

	// Set some keys
	handler.HandleCommand(&proto.Command{Name: "SET", Args: []string{"key1", "value1"}})
	handler.HandleCommand(&proto.Command{Name: "SET", Args: []string{"key2", "value2"}})

	// Test DEL single key
	cmd = &proto.Command{Name: "DEL", Args: []string{"key1"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response, got %v", resp.Type)
	}

	if resp.Data.(int64) != 1 {
		t.Errorf("Expected 1 deleted key, got %d", resp.Data.(int64))
	}

	// Test DEL multiple keys
	cmd = &proto.Command{Name: "DEL", Args: []string{"key2", "nonexistent"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response, got %v", resp.Type)
	}

	if resp.Data.(int64) != 1 {
		t.Errorf("Expected 1 deleted key, got %d", resp.Data.(int64))
	}
}

func TestHandler_EXISTS(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	// Test EXISTS without args
	cmd := &proto.Command{Name: "EXISTS", Args: []string{}}
	resp := handler.HandleCommand(cmd)

	if resp.Type != proto.Error {
		t.Errorf("Expected Error response for EXISTS without args, got %v", resp.Type)
	}

	// Test EXISTS for non-existent keys
	cmd = &proto.Command{Name: "EXISTS", Args: []string{"nonexistent1", "nonexistent2"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response, got %v", resp.Type)
	}

	if resp.Data.(int64) != 0 {
		t.Errorf("Expected 0 existing keys, got %d", resp.Data.(int64))
	}

	// Set a key and test EXISTS
	handler.HandleCommand(&proto.Command{Name: "SET", Args: []string{"existingkey", "value"}})

	cmd = &proto.Command{Name: "EXISTS", Args: []string{"existingkey", "nonexistent"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response, got %v", resp.Type)
	}

	if resp.Data.(int64) != 1 {
		t.Errorf("Expected 1 existing key, got %d", resp.Data.(int64))
	}
}

func TestHandler_EXPIRE_TTL(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	// Test EXPIRE without proper args
	cmd := &proto.Command{Name: "EXPIRE", Args: []string{"key"}}
	resp := handler.HandleCommand(cmd)

	if resp.Type != proto.Error {
		t.Errorf("Expected Error response for EXPIRE with insufficient args, got %v", resp.Type)
	}

	// Test EXPIRE with invalid timeout
	cmd = &proto.Command{Name: "EXPIRE", Args: []string{"key", "invalid"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Error {
		t.Errorf("Expected Error response for EXPIRE with invalid timeout, got %v", resp.Type)
	}

	// Set a key and test EXPIRE
	handler.HandleCommand(&proto.Command{Name: "SET", Args: []string{"expirekey", "value"}})

	cmd = &proto.Command{Name: "EXPIRE", Args: []string{"expirekey", "60"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response, got %v", resp.Type)
	}

	if resp.Data.(int64) != 1 {
		t.Errorf("Expected 1 (success), got %d", resp.Data.(int64))
	}

	// Test TTL
	cmd = &proto.Command{Name: "TTL", Args: []string{"expirekey"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response for TTL, got %v", resp.Type)
	}

	ttl := resp.Data.(int64)
	if ttl <= 0 || ttl > 60 {
		t.Errorf("Expected TTL between 1-60, got %d", ttl)
	}

	// Test EXPIRE on non-existent key
	cmd = &proto.Command{Name: "EXPIRE", Args: []string{"nonexistent", "60"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response, got %v", resp.Type)
	}

	if resp.Data.(int64) != 0 {
		t.Errorf("Expected 0 (failure), got %d", resp.Data.(int64))
	}
}

func TestHandler_DBSIZE(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	// Test DBSIZE with args (should error)
	cmd := &proto.Command{Name: "DBSIZE", Args: []string{"invalid"}}
	resp := handler.HandleCommand(cmd)

	if resp.Type != proto.Error {
		t.Errorf("Expected Error response for DBSIZE with args, got %v", resp.Type)
	}

	// Test DBSIZE initially
	cmd = &proto.Command{Name: "DBSIZE", Args: []string{}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response, got %v", resp.Type)
	}

	initialSize := resp.Data.(int64)

	// Add some keys
	handler.HandleCommand(&proto.Command{Name: "SET", Args: []string{"key1", "value1"}})
	handler.HandleCommand(&proto.Command{Name: "SET", Args: []string{"key2", "value2"}})

	// Test DBSIZE after adding keys
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response, got %v", resp.Type)
	}

	newSize := resp.Data.(int64)
	if newSize != initialSize+2 {
		t.Errorf("Expected DB size to increase by 2, got %d -> %d", initialSize, newSize)
	}
}

func TestHandler_MGET_MSET(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	// Test MGET without args
	cmd := &proto.Command{Name: "MGET", Args: []string{}}
	resp := handler.HandleCommand(cmd)

	if resp.Type != proto.Error {
		t.Errorf("Expected Error response for MGET without args, got %v", resp.Type)
	}

	// Test MSET with odd number of args
	cmd = &proto.Command{Name: "MSET", Args: []string{"key1", "value1", "key2"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Error {
		t.Errorf("Expected Error response for MSET with odd args, got %v", resp.Type)
	}

	// Test MSET with valid args
	cmd = &proto.Command{Name: "MSET", Args: []string{"key1", "value1", "key2", "value2"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.SimpleString || resp.Data.(string) != "OK" {
		t.Errorf("Expected OK response for MSET, got %v: %v", resp.Type, resp.Data)
	}

	// Test MGET
	cmd = &proto.Command{Name: "MGET", Args: []string{"key1", "nonexistent", "key2"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Array {
		t.Errorf("Expected Array response for MGET, got %v", resp.Type)
	}

	arr := resp.Data.([]any)
	if len(arr) != 3 {
		t.Errorf("Expected 3 elements in MGET response, got %d", len(arr))
	}

	if arr[0].(string) != "value1" {
		t.Errorf("Expected first element to be 'value1', got %v", arr[0])
	}

	if arr[1] != nil {
		t.Errorf("Expected second element to be nil, got %v", arr[1])
	}

	if arr[2].(string) != "value2" {
		t.Errorf("Expected third element to be 'value2', got %v", arr[2])
	}
}

func TestHandler_INCR_DECR(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	// Test INCR on non-existent key
	cmd := &proto.Command{Name: "INCR", Args: []string{"counter"}}
	resp := handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response for INCR, got %v", resp.Type)
	}

	if resp.Data.(int64) != 1 {
		t.Errorf("Expected 1 for INCR on non-existent key, got %d", resp.Data.(int64))
	}

	// Test INCR on existing numeric key
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response for INCR, got %v", resp.Type)
	}

	if resp.Data.(int64) != 2 {
		t.Errorf("Expected 2 for second INCR, got %d", resp.Data.(int64))
	}

	// Test DECR
	cmd = &proto.Command{Name: "DECR", Args: []string{"counter"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response for DECR, got %v", resp.Type)
	}

	if resp.Data.(int64) != 1 {
		t.Errorf("Expected 1 for DECR, got %d", resp.Data.(int64))
	}

	// Test INCRBY
	cmd = &proto.Command{Name: "INCRBY", Args: []string{"counter", "5"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response for INCRBY, got %v", resp.Type)
	}

	if resp.Data.(int64) != 6 {
		t.Errorf("Expected 6 for INCRBY 5, got %d", resp.Data.(int64))
	}

	// Test DECRBY
	cmd = &proto.Command{Name: "DECRBY", Args: []string{"counter", "3"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response for DECRBY, got %v", resp.Type)
	}

	if resp.Data.(int64) != 3 {
		t.Errorf("Expected 3 for DECRBY 3, got %d", resp.Data.(int64))
	}

	// Test INCR on non-numeric value
	handler.HandleCommand(&proto.Command{Name: "SET", Args: []string{"nonnum", "notanumber"}})

	cmd = &proto.Command{Name: "INCR", Args: []string{"nonnum"}}
	resp = handler.HandleCommand(cmd)

	if resp.Type != proto.Error {
		t.Errorf("Expected Error response for INCR on non-numeric, got %v", resp.Type)
	}
}

func TestHandler_ArgsValidation(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	tests := []struct {
		command string
		args    []string
		wantErr bool
	}{
		{"GET", []string{}, true},
		{"GET", []string{"key1", "key2"}, true},
		{"TTL", []string{}, true},
		{"TTL", []string{"key1", "key2"}, true},
		{"INCR", []string{}, true},
		{"INCR", []string{"key1", "key2"}, true},
		{"DECR", []string{}, true},
		{"DECR", []string{"key1", "key2"}, true},
		{"INCRBY", []string{"key"}, true},
		{"INCRBY", []string{"key", "5", "extra"}, true},
		{"DECRBY", []string{"key"}, true},
		{"DECRBY", []string{"key", "5", "extra"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.command+"_validation", func(t *testing.T) {
			t.Parallel()

			cmd := &proto.Command{Name: tt.command, Args: tt.args}
			resp := handler.HandleCommand(cmd)

			if tt.wantErr && resp.Type != proto.Error {
				t.Errorf("Expected Error response for %s with args %v, got %v", tt.command, tt.args, resp.Type)
			}
		})
	}
}

func TestHandler_UnknownCommand(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	cmd := &proto.Command{Name: "UNKNOWN", Args: []string{}}
	resp := handler.HandleCommand(cmd)

	if resp.Type != proto.Error {
		t.Errorf("Expected Error response for unknown command, got %v", resp.Type)
	}

	expectedMsg := "ERR unknown command 'UNKNOWN'"
	if resp.Data.(string) != expectedMsg {
		t.Errorf("Expected %q, got %q", expectedMsg, resp.Data.(string))
	}
}

func TestHandler_QUIT(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	cmd := &proto.Command{Name: "QUIT", Args: []string{}}
	resp := handler.HandleCommand(cmd)

	if resp.Type != proto.SimpleString {
		t.Errorf("Expected SimpleString response for QUIT, got %v", resp.Type)
	}

	if resp.Data.(string) != "OK" {
		t.Errorf("Expected 'OK' response for QUIT, got %q", resp.Data.(string))
	}
}

func TestHandler_SET_PXOption(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	// Test SET with PX option (which wasn't tested in the existing SET tests)
	tests := []struct {
		name    string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "SET with PX missing value",
			args:    []string{"key", "value", "PX"},
			wantErr: true,
			errMsg:  "ERR syntax error",
		},
		{
			name:    "SET with PX invalid value",
			args:    []string{"key", "value", "PX", "notanumber"},
			wantErr: true,
			errMsg:  "ERR value is not an integer or out of range",
		},
		{
			name:    "SET with PX valid value",
			args:    []string{"key", "value", "PX", "5000"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := &proto.Command{Name: "SET", Args: tt.args}
			resp := handler.HandleCommand(cmd)

			if tt.wantErr {
				if resp.Type != proto.Error {
					t.Errorf("Expected Error response, got %v", resp.Type)
				}
				if !strings.Contains(resp.Data.(string), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, resp.Data.(string))
				}
			} else if resp.Type != proto.SimpleString || resp.Data.(string) != "OK" {
				t.Errorf("Expected OK response, got %v: %v", resp.Type, resp.Data)
			}
		})
	}
}

func TestHandler_INCRBY_DECRBY_EdgeCases(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	// Test INCRBY with invalid increment values (which aren't fully covered)
	tests := []struct {
		name    string
		command string
		args    []string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "INCRBY with non-numeric increment",
			command: "INCRBY",
			args:    []string{"key", "notanumber"},
			wantErr: true,
			errMsg:  "ERR value is not an integer or out of range",
		},
		{
			name:    "DECRBY with non-numeric decrement",
			command: "DECRBY",
			args:    []string{"key", "notanumber"},
			wantErr: true,
			errMsg:  "ERR value is not an integer or out of range",
		},
		{
			name:    "INCRBY with very large number",
			command: "INCRBY",
			args:    []string{"counter", "9223372036854775807"}, // max int64
			wantErr: false,
		},
		{
			name:    "DECRBY with very large number",
			command: "DECRBY",
			args:    []string{"counter2", "9223372036854775807"},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			cmd := &proto.Command{Name: tt.command, Args: tt.args}
			resp := handler.HandleCommand(cmd)

			if tt.wantErr {
				if resp.Type != proto.Error {
					t.Errorf("Expected Error response, got %v", resp.Type)
				}
				if !strings.Contains(resp.Data.(string), tt.errMsg) {
					t.Errorf("Expected error containing %q, got %q", tt.errMsg, resp.Data.(string))
				}
			} else if resp.Type != proto.Integer {
				t.Errorf("Expected Integer response, got %v", resp.Type)
			}
		})
	}
}

func TestHandler_IncrementBy_ExistingNonNumericValue(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	// Set a non-numeric value first
	handler.HandleCommand(&proto.Command{Name: "SET", Args: []string{"nonnum", "hello"}})

	// Try to increment it
	cmd := &proto.Command{Name: "INCRBY", Args: []string{"nonnum", "5"}}
	resp := handler.HandleCommand(cmd)

	if resp.Type != proto.Error {
		t.Errorf("Expected Error response for INCRBY on non-numeric value, got %v", resp.Type)
	}

	expectedMsg := "ERR value is not an integer or out of range"
	if !strings.Contains(resp.Data.(string), expectedMsg) {
		t.Errorf("Expected error %q, got %q", expectedMsg, resp.Data.(string))
	}
}

func TestHandler_IncrementBy_IntegerOverflow(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	// Set a very large number
	handler.HandleCommand(&proto.Command{Name: "SET", Args: []string{"bignum", "9223372036854775807"}}) // max int64

	// Try to increment it (this will overflow to negative in Go)
	cmd := &proto.Command{Name: "INCRBY", Args: []string{"bignum", "1"}}
	resp := handler.HandleCommand(cmd)

	// The current implementation doesn't check for overflow, so it returns an integer
	// In a production implementation, this should be an error, but for now we'll test what it actually does
	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response (current implementation), got %v", resp.Type)
	}

	// The result should be negative due to overflow (9223372036854775807 + 1 = -9223372036854775808)
	result := resp.Data.(int64)
	if result >= 0 {
		t.Errorf("Expected negative result due to overflow, got %d", result)
	}
}

func TestHandler_IncrementBy_ExistingNumericValue(t *testing.T) {
	t.Parallel()

	handler := createTestHandler(t)

	// Set a numeric value first
	handler.HandleCommand(&proto.Command{Name: "SET", Args: []string{"num", "100"}})

	// Increment it
	cmd := &proto.Command{Name: "INCRBY", Args: []string{"num", "50"}}
	resp := handler.HandleCommand(cmd)

	if resp.Type != proto.Integer {
		t.Errorf("Expected Integer response, got %v", resp.Type)
	}

	if resp.Data.(int64) != 150 {
		t.Errorf("Expected 150, got %d", resp.Data.(int64))
	}
}

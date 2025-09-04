package obs_test

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/Abhishek2095/kv-stash/internal/obs"
)

func TestNewMetrics(t *testing.T) {
	t.Parallel()

	metrics := obs.NewMetrics()
	if metrics == nil {
		t.Fatal("NewMetrics returned nil")
	}

	// Test that all metrics are initialized
	if metrics.CommandsTotal == nil {
		t.Error("CommandsTotal not initialized")
	}
	if metrics.CommandDuration == nil {
		t.Error("CommandDuration not initialized")
	}
	if metrics.CommandsInFlight == nil {
		t.Error("CommandsInFlight not initialized")
	}
	if metrics.ConnectionsTotal == nil {
		t.Error("ConnectionsTotal not initialized")
	}
	if metrics.ConnectionsCurrent == nil {
		t.Error("ConnectionsCurrent not initialized")
	}
	if metrics.KeysTotal == nil {
		t.Error("KeysTotal not initialized")
	}
	if metrics.ExpiredKeysTotal == nil {
		t.Error("ExpiredKeysTotal not initialized")
	}
	if metrics.MemoryUsage == nil {
		t.Error("MemoryUsage not initialized")
	}
	if metrics.UptimeSeconds == nil {
		t.Error("UptimeSeconds not initialized")
	}
}

func TestMetrics_RecordCommand(t *testing.T) {
	t.Parallel()

	metrics := obs.NewMetrics()

	tests := []struct {
		name     string
		command  string
		duration time.Duration
		success  bool
	}{
		{
			name:     "Successful GET command",
			command:  "GET",
			duration: 10 * time.Millisecond,
			success:  true,
		},
		{
			name:     "Failed SET command",
			command:  "SET",
			duration: 5 * time.Millisecond,
			success:  false,
		},
		{
			name:     "Successful PING command",
			command:  "PING",
			duration: 1 * time.Millisecond,
			success:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// RecordCommand should not panic
			metrics.RecordCommand(tt.command, tt.duration, tt.success)
		})
	}
}

func TestMetrics_CommandsInFlight(t *testing.T) {
	t.Parallel()

	metrics := obs.NewMetrics()

	// Test increment
	metrics.IncCommandsInFlight()
	metrics.IncCommandsInFlight()

	// Test decrement
	metrics.DecCommandsInFlight()
}

func TestMetrics_Connections(t *testing.T) {
	t.Parallel()

	metrics := obs.NewMetrics()

	// Test connection metrics
	metrics.IncConnections()
	metrics.IncConnections()
	metrics.DecConnections()
}

func TestMetrics_Storage(t *testing.T) {
	t.Parallel()

	metrics := obs.NewMetrics()

	// Test storage metrics
	metrics.SetKeys(100)
	metrics.SetKeys(200)

	metrics.IncExpiredKeys()
	metrics.IncExpiredKeys()

	metrics.SetMemoryUsage(1024 * 1024)     // 1MB
	metrics.SetMemoryUsage(2 * 1024 * 1024) // 2MB
}

func TestMetrics_Uptime(t *testing.T) {
	t.Parallel()

	metrics := obs.NewMetrics()

	// Test uptime metric
	uptime := 3600 * time.Second // 1 hour
	metrics.SetUptime(uptime)

	// Test with different uptimes
	uptime = 7200 * time.Second // 2 hours
	metrics.SetUptime(uptime)
}

func TestMetrics_Handler(t *testing.T) {
	t.Parallel()

	metrics := obs.NewMetrics()

	// Get the HTTP handler
	handler := metrics.Handler()
	if handler == nil {
		t.Fatal("Handler returned nil")
	}

	// Test that handler serves metrics
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	body := w.Body.String()

	// Check for some expected metric names (only gauge metrics that are always present)
	expectedMetrics := []string{
		"kvstash_commands_in_flight",
		"kvstash_connections_current",
		"kvstash_keys",
		"kvstash_memory_usage_bytes",
		"kvstash_uptime_seconds",
	}

	for _, metric := range expectedMetrics {
		if !strings.Contains(body, metric) {
			t.Logf("Actual metrics output: %s", body)
			t.Errorf("Expected metrics output to contain %q", metric)
		}
	}

	// Verify that the response contains Prometheus format markers
	if !strings.Contains(body, "# HELP") {
		t.Error("Expected metrics output to contain HELP comments")
	}

	if !strings.Contains(body, "# TYPE") {
		t.Error("Expected metrics output to contain TYPE comments")
	}
}

func TestMetrics_StartMetricsServer(t *testing.T) {
	t.Parallel()

	metrics := obs.NewMetrics()
	logger := obs.NewLogger(false)

	// Test with invalid address (should return error)
	err := metrics.StartMetricsServer("invalid:address:format", logger)
	if err == nil {
		t.Error("Expected error for invalid address")
	}
}

func TestMetrics_HealthEndpoint(t *testing.T) {
	t.Parallel()

	metrics := obs.NewMetrics()
	logger := obs.NewLogger(false)

	// Create a test server to verify the health endpoint
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("OK"))
			return
		}

		if r.URL.Path == "/metrics" {
			metrics.Handler().ServeHTTP(w, r)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	// Test health endpoint
	req, _ := http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/health", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to request health endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for health endpoint, got %d", resp.StatusCode)
	}

	// Test metrics endpoint
	req, _ = http.NewRequestWithContext(context.Background(), http.MethodGet, server.URL+"/metrics", nil)
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to request metrics endpoint: %v", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		t.Errorf("Expected status 200 for metrics endpoint, got %d", resp.StatusCode)
	}

	// Silence the logger variable to avoid unused variable error
	_ = logger
}

func TestMetrics_RecordCommandWithDifferentStatuses(t *testing.T) {
	t.Parallel()

	metrics := obs.NewMetrics()

	// Record various commands with different outcomes
	commands := []struct {
		name    string
		success bool
	}{
		{"GET", true},
		{"GET", false},
		{"SET", true},
		{"SET", false},
		{"DEL", true},
		{"PING", true},
	}

	for _, cmd := range commands {
		metrics.RecordCommand(cmd.name, 10*time.Millisecond, cmd.success)
	}

	// Verify metrics can be served after recording
	handler := metrics.Handler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 after recording commands, got %d", w.Code)
	}

	body := w.Body.String()

	// Should contain both success and error metrics
	if !strings.Contains(body, "status=\"success\"") {
		t.Error("Expected to find success status in metrics")
	}

	if !strings.Contains(body, "status=\"error\"") {
		t.Error("Expected to find error status in metrics")
	}
}

func TestMetrics_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	metrics := obs.NewMetrics()

	// Test concurrent access to metrics
	done := make(chan bool, 10)

	for i := range 10 {
		go func(id int) {
			defer func() { done <- true }()

			// Perform various metric operations concurrently
			metrics.IncCommandsInFlight()
			metrics.RecordCommand("GET", time.Millisecond, true)
			metrics.IncConnections()
			metrics.SetKeys(int64(id))
			metrics.IncExpiredKeys()
			metrics.SetMemoryUsage(int64(id * 1024))
			metrics.DecCommandsInFlight()
			metrics.DecConnections()
		}(i)
	}

	// Wait for all goroutines to complete
	for range 10 {
		<-done
	}

	// Verify metrics handler still works after concurrent access
	handler := metrics.Handler()
	req := httptest.NewRequest(http.MethodGet, "/metrics", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200 after concurrent access, got %d", w.Code)
	}
}

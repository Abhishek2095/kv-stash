package server_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/Abhishek2095/kv-stash/internal/server"
)

func TestDefaultConfig(t *testing.T) {
	t.Parallel()

	config := server.DefaultConfig()
	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	// Test server defaults
	if config.Server.ListenAddr != ":6380" {
		t.Errorf("Expected default listen addr ':6380', got %q", config.Server.ListenAddr)
	}

	if config.Server.Shards != 8 {
		t.Errorf("Expected default shards 8, got %d", config.Server.Shards)
	}

	if config.Server.AuthPassword != "" {
		t.Errorf("Expected empty default auth password, got %q", config.Server.AuthPassword)
	}

	if config.Server.ReadTimeout != 30*time.Second {
		t.Errorf("Expected default read timeout 30s, got %v", config.Server.ReadTimeout)
	}

	if config.Server.WriteTimeout != 30*time.Second {
		t.Errorf("Expected default write timeout 30s, got %v", config.Server.WriteTimeout)
	}

	// Test limits defaults
	if config.Limits.MaxClients != 10000 {
		t.Errorf("Expected default max clients 10000, got %d", config.Limits.MaxClients)
	}

	if config.Limits.MaxPipeline != 1024 {
		t.Errorf("Expected default max pipeline 1024, got %d", config.Limits.MaxPipeline)
	}

	// Test storage defaults
	if config.Storage.MaxMemoryBytes != 0 {
		t.Errorf("Expected default max memory 0 (unlimited), got %d", config.Storage.MaxMemoryBytes)
	}

	if config.Storage.EvictionPolicy != "noeviction" {
		t.Errorf("Expected default eviction policy 'noeviction', got %q", config.Storage.EvictionPolicy)
	}

	// Test TTL defaults
	if config.TTL.Strategy != "lazy+active" {
		t.Errorf("Expected default TTL strategy 'lazy+active', got %q", config.TTL.Strategy)
	}

	if config.TTL.ActiveCycle != 50*time.Millisecond {
		t.Errorf("Expected default active cycle 50ms, got %v", config.TTL.ActiveCycle)
	}

	// Test persistence defaults
	if config.Persistence.Snapshot.Enabled {
		t.Error("Expected snapshot to be disabled by default")
	}

	if config.Persistence.Snapshot.IntervalSeconds != 300 {
		t.Errorf("Expected default snapshot interval 300s, got %d", config.Persistence.Snapshot.IntervalSeconds)
	}

	if config.Persistence.AOF.Enabled {
		t.Error("Expected AOF to be disabled by default")
	}

	if config.Persistence.AOF.Fsync != "everysec" {
		t.Errorf("Expected default AOF fsync 'everysec', got %q", config.Persistence.AOF.Fsync)
	}

	// Test replication defaults
	if config.Replication.Role != "leader" {
		t.Errorf("Expected default replication role 'leader', got %q", config.Replication.Role)
	}

	// Test observability defaults
	if config.Observability.LogLevel != "info" {
		t.Errorf("Expected default log level 'info', got %q", config.Observability.LogLevel)
	}

	if config.Observability.PrometheusListen != ":9100" {
		t.Errorf("Expected default Prometheus listen ':9100', got %q", config.Observability.PrometheusListen)
	}
}

func TestLoadConfig_NonExistentFile(t *testing.T) {
	t.Parallel()

	config, err := server.LoadConfig("/nonexistent/path/config.yml")
	if err != nil {
		t.Errorf("LoadConfig should not error for non-existent file, got: %v", err)
	}

	if config == nil {
		t.Fatal("LoadConfig returned nil config for non-existent file")
	}

	// Should return default config
	defaultConfig := server.DefaultConfig()
	if config.Server.ListenAddr != defaultConfig.Server.ListenAddr {
		t.Error("Non-existent file should return default config")
	}
}

func TestLoadConfig_ValidYAML(t *testing.T) {
	t.Parallel()

	// Create temporary config file
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.yml")

	configContent := `
server:
  listen_addr: ":7000"
  shards: 16
  auth_password: "secret"
  read_timeout: 60s
  write_timeout: 60s

limits:
  max_clients: 5000
  max_pipeline: 512

storage:
  maxmemory_bytes: 1073741824
  eviction_policy: "allkeys-lru"

ttl:
  strategy: "lazy"
  active_cycle_ms: 100ms

persistence:
  snapshot:
    enabled: true
    interval_seconds: 600
    dir: "/data/snapshots"
  aof:
    enabled: true
    fsync: "always"
    dir: "/data/aof"

replication:
  role: "follower"
  leader_addr: "leader:6380"

observability:
  log_level: "debug"
  prometheus_listen: ":9200"
  otlp_endpoint: "http://jaeger:14268/api/traces"
`

	err := os.WriteFile(configFile, []byte(configContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	config, err := server.LoadConfig(configFile)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify parsed values
	if config.Server.ListenAddr != ":7000" {
		t.Errorf("Expected listen addr ':7000', got %q", config.Server.ListenAddr)
	}

	if config.Server.Shards != 16 {
		t.Errorf("Expected shards 16, got %d", config.Server.Shards)
	}

	if config.Server.AuthPassword != "secret" {
		t.Errorf("Expected auth password 'secret', got %q", config.Server.AuthPassword)
	}

	if config.Limits.MaxClients != 5000 {
		t.Errorf("Expected max clients 5000, got %d", config.Limits.MaxClients)
	}

	if config.Storage.EvictionPolicy != "allkeys-lru" {
		t.Errorf("Expected eviction policy 'allkeys-lru', got %q", config.Storage.EvictionPolicy)
	}

	if !config.Persistence.Snapshot.Enabled {
		t.Error("Expected snapshot to be enabled")
	}

	if config.Replication.Role != "follower" {
		t.Errorf("Expected replication role 'follower', got %q", config.Replication.Role)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid.yml")

	invalidContent := `
server:
  listen_addr: ":6380"
  invalid_yaml: [unclosed bracket
`

	err := os.WriteFile(configFile, []byte(invalidContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err = server.LoadConfig(configFile)
	if err == nil {
		t.Error("LoadConfig should error for invalid YAML")
	}

	if !strings.Contains(err.Error(), "failed to parse config file") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestValidate(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		modify    func(*server.AppConfig)
		wantErr   bool
		errString string
	}{
		{
			name: "Valid config",
			modify: func(_ *server.AppConfig) {
				// Default config should be valid
			},
			wantErr: false,
		},
		{
			name: "Zero shards",
			modify: func(c *server.AppConfig) {
				c.Server.Shards = 0
			},
			wantErr:   true,
			errString: "server.shards must be greater than 0",
		},
		{
			name: "Negative shards",
			modify: func(c *server.AppConfig) {
				c.Server.Shards = -1
			},
			wantErr:   true,
			errString: "server.shards must be greater than 0",
		},
		{
			name: "Zero max clients",
			modify: func(c *server.AppConfig) {
				c.Limits.MaxClients = 0
			},
			wantErr:   true,
			errString: "limits.max_clients must be greater than 0",
		},
		{
			name: "Negative max clients",
			modify: func(c *server.AppConfig) {
				c.Limits.MaxClients = -1
			},
			wantErr:   true,
			errString: "limits.max_clients must be greater than 0",
		},
		{
			name: "Zero max pipeline",
			modify: func(c *server.AppConfig) {
				c.Limits.MaxPipeline = 0
			},
			wantErr:   true,
			errString: "limits.max_pipeline must be greater than 0",
		},
		{
			name: "Invalid eviction policy",
			modify: func(c *server.AppConfig) {
				c.Storage.EvictionPolicy = "invalid-policy"
			},
			wantErr:   true,
			errString: "invalid eviction policy",
		},
		{
			name: "Valid allkeys-lru eviction policy",
			modify: func(c *server.AppConfig) {
				c.Storage.EvictionPolicy = "allkeys-lru"
			},
			wantErr: false,
		},
		{
			name: "Valid volatile-lfu eviction policy",
			modify: func(c *server.AppConfig) {
				c.Storage.EvictionPolicy = "volatile-lfu"
			},
			wantErr: false,
		},
		{
			name: "Invalid AOF fsync policy",
			modify: func(c *server.AppConfig) {
				c.Persistence.AOF.Fsync = "invalid-fsync"
			},
			wantErr:   true,
			errString: "invalid AOF fsync policy",
		},
		{
			name: "Valid AOF fsync always",
			modify: func(c *server.AppConfig) {
				c.Persistence.AOF.Fsync = "always"
			},
			wantErr: false,
		},
		{
			name: "Valid AOF fsync no",
			modify: func(c *server.AppConfig) {
				c.Persistence.AOF.Fsync = "no"
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			config := server.DefaultConfig()
			tt.modify(config)

			err := config.Validate()

			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() should have returned error for %s", tt.name)
					return
				}
				if !strings.Contains(err.Error(), tt.errString) {
					t.Errorf("Expected error containing %q, got %q", tt.errString, err.Error())
				}
			} else if err != nil {
				t.Errorf("Validate() should not have returned error for %s: %v", tt.name, err)
			}
		})
	}
}

func TestLoadConfig_InvalidConfig(t *testing.T) {
	t.Parallel()

	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "invalid_config.yml")

	// Create config with invalid values
	invalidContent := `
server:
  shards: 0  # Invalid: must be > 0

limits:
  max_clients: -1  # Invalid: must be > 0
`

	err := os.WriteFile(configFile, []byte(invalidContent), 0o600)
	if err != nil {
		t.Fatalf("Failed to write config file: %v", err)
	}

	_, err = server.LoadConfig(configFile)
	if err == nil {
		t.Error("LoadConfig should error for invalid config values")
	}

	if !strings.Contains(err.Error(), "invalid configuration") {
		t.Errorf("Expected configuration validation error, got: %v", err)
	}
}

func TestLoadConfig_FileReadError(t *testing.T) {
	t.Parallel()

	// Try to load a directory as config file (should cause read error)
	tmpDir := t.TempDir()

	_, err := server.LoadConfig(tmpDir)
	if err == nil {
		t.Error("LoadConfig should error when trying to read directory")
	}

	if !strings.Contains(err.Error(), "failed to read config file") {
		t.Errorf("Expected file read error, got: %v", err)
	}
}

func TestConfigStructTypes(t *testing.T) {
	t.Parallel()

	config := server.DefaultConfig()

	// Test that all config struct fields have expected types
	if config.Server.ReadTimeout.String() != "30s" {
		t.Errorf("Expected ReadTimeout to be 30s, got %v", config.Server.ReadTimeout)
	}

	if config.TTL.ActiveCycle.String() != "50ms" {
		t.Errorf("Expected ActiveCycle to be 50ms, got %v", config.TTL.ActiveCycle)
	}

	// Test integer fields
	if config.Persistence.Snapshot.IntervalSeconds != 300 {
		t.Errorf("Expected IntervalSeconds to be int 300, got %d", config.Persistence.Snapshot.IntervalSeconds)
	}

	// Test string fields
	if len(config.Persistence.Snapshot.Dir) == 0 {
		t.Error("Expected Dir to be non-empty string")
	}

	// Test boolean fields
	if config.Persistence.Snapshot.Enabled {
		t.Error("Expected Enabled to be false by default")
	}
}

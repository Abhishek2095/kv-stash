// Package server provides the core server implementation for the kv-stash Redis-compatible key-value store.
package server

import (
	"errors"
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

const (
	// Default server configuration constants
	defaultShardCount           = 8
	defaultReadTimeoutSeconds   = 30
	defaultWriteTimeoutSeconds  = 30
	defaultMaxClients           = 10000
	defaultMaxPipeline          = 1024
	defaultActiveCycleMs        = 50
	defaultSnapshotIntervalSecs = 300
)

// AppConfig represents the application configuration
type AppConfig struct {
	Server        Config              `yaml:"server"`
	Limits        LimitsConfig        `yaml:"limits"`
	Storage       StorageConfig       `yaml:"storage"`
	TTL           TTLConfig           `yaml:"ttl"`
	Persistence   PersistenceConfig   `yaml:"persistence"`
	Replication   ReplicationConfig   `yaml:"replication"`
	Observability ObservabilityConfig `yaml:"observability"`
}

// Config contains server-specific settings
type Config struct {
	ListenAddr   string        `yaml:"listen_addr"`
	Shards       int           `yaml:"shards"`
	AuthPassword string        `yaml:"auth_password"`
	ReadTimeout  time.Duration `yaml:"read_timeout"`
	WriteTimeout time.Duration `yaml:"write_timeout"`
}

// LimitsConfig contains connection and pipeline limits
type LimitsConfig struct {
	MaxClients  int `yaml:"max_clients"`
	MaxPipeline int `yaml:"max_pipeline"`
}

// StorageConfig contains storage-related settings
type StorageConfig struct {
	MaxMemoryBytes int64  `yaml:"maxmemory_bytes"`
	EvictionPolicy string `yaml:"eviction_policy"`
}

// TTLConfig contains TTL-related settings
type TTLConfig struct {
	Strategy    string        `yaml:"strategy"`
	ActiveCycle time.Duration `yaml:"active_cycle_ms"`
}

// PersistenceConfig contains persistence settings
type PersistenceConfig struct {
	Snapshot SnapshotConfig `yaml:"snapshot"`
	AOF      AOFConfig      `yaml:"aof"`
}

// SnapshotConfig contains snapshot-specific settings
type SnapshotConfig struct {
	Enabled         bool   `yaml:"enabled"`
	IntervalSeconds int    `yaml:"interval_seconds"`
	Dir             string `yaml:"dir"`
}

// AOFConfig contains AOF-specific settings
type AOFConfig struct {
	Enabled bool   `yaml:"enabled"`
	Fsync   string `yaml:"fsync"`
	Dir     string `yaml:"dir"`
}

// ReplicationConfig contains replication settings
type ReplicationConfig struct {
	Role       string `yaml:"role"`
	LeaderAddr string `yaml:"leader_addr"`
}

// ObservabilityConfig contains observability settings
type ObservabilityConfig struct {
	LogLevel         string `yaml:"log_level"`
	PrometheusListen string `yaml:"prometheus_listen"`
	OTLPEndpoint     string `yaml:"otlp_endpoint"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *AppConfig {
	return &AppConfig{
		Server: Config{
			ListenAddr:   ":6380",
			Shards:       defaultShardCount,
			AuthPassword: "",
			ReadTimeout:  defaultReadTimeoutSeconds * time.Second,
			WriteTimeout: defaultWriteTimeoutSeconds * time.Second,
		},
		Limits: LimitsConfig{
			MaxClients:  defaultMaxClients,
			MaxPipeline: defaultMaxPipeline,
		},
		Storage: StorageConfig{
			MaxMemoryBytes: 0, // unlimited
			EvictionPolicy: "noeviction",
		},
		TTL: TTLConfig{
			Strategy:    "lazy+active",
			ActiveCycle: defaultActiveCycleMs * time.Millisecond,
		},
		Persistence: PersistenceConfig{
			Snapshot: SnapshotConfig{
				Enabled:         false,
				IntervalSeconds: defaultSnapshotIntervalSecs,
				Dir:             "./data",
			},
			AOF: AOFConfig{
				Enabled: false,
				Fsync:   "everysec",
				Dir:     "./data",
			},
		},
		Replication: ReplicationConfig{
			Role:       "leader",
			LeaderAddr: "",
		},
		Observability: ObservabilityConfig{
			LogLevel:         "info",
			PrometheusListen: ":9100",
			OTLPEndpoint:     "",
		},
	}
}

// LoadConfig loads configuration from a file
func LoadConfig(path string) (*AppConfig, error) {
	cfg := DefaultConfig()

	// If config file doesn't exist, return default config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	data, err := os.ReadFile(path) // #nosec G304 -- Path is validated by caller
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return cfg, nil
}

// Validate validates the configuration
func (c *AppConfig) Validate() error {
	if c.Server.Shards <= 0 {
		return errors.New("server.shards must be greater than 0")
	}

	if c.Limits.MaxClients <= 0 {
		return errors.New("limits.max_clients must be greater than 0")
	}

	if c.Limits.MaxPipeline <= 0 {
		return errors.New("limits.max_pipeline must be greater than 0")
	}

	validEvictionPolicies := map[string]bool{
		"noeviction":   true,
		"allkeys-lru":  true,
		"volatile-lru": true,
		"allkeys-lfu":  true,
		"volatile-lfu": true,
	}
	if !validEvictionPolicies[c.Storage.EvictionPolicy] {
		return fmt.Errorf("invalid eviction policy: %s", c.Storage.EvictionPolicy)
	}

	validFsyncPolicies := map[string]bool{
		"always":   true,
		"everysec": true,
		"no":       true,
	}
	if !validFsyncPolicies[c.Persistence.AOF.Fsync] {
		return fmt.Errorf("invalid AOF fsync policy: %s", c.Persistence.AOF.Fsync)
	}

	return nil
}

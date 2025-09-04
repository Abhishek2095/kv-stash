// Copyright (c) 2024 Abhishek2095
// SPDX-License-Identifier: MIT

package server

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config represents the server configuration
type Config struct {
	Server        ServerConfig        `yaml:"server"`
	Limits        LimitsConfig        `yaml:"limits"`
	Storage       StorageConfig       `yaml:"storage"`
	TTL           TTLConfig           `yaml:"ttl"`
	Persistence   PersistenceConfig   `yaml:"persistence"`
	Replication   ReplicationConfig   `yaml:"replication"`
	Observability ObservabilityConfig `yaml:"observability"`
}

// ServerConfig contains server-specific settings
type ServerConfig struct {
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
	Strategy      string        `yaml:"strategy"`
	ActiveCycleMs time.Duration `yaml:"active_cycle_ms"`
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
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			ListenAddr:   ":6380",
			Shards:       8,
			AuthPassword: "",
			ReadTimeout:  30 * time.Second,
			WriteTimeout: 30 * time.Second,
		},
		Limits: LimitsConfig{
			MaxClients:  10000,
			MaxPipeline: 1024,
		},
		Storage: StorageConfig{
			MaxMemoryBytes: 0, // unlimited
			EvictionPolicy: "noeviction",
		},
		TTL: TTLConfig{
			Strategy:      "lazy+active",
			ActiveCycleMs: 50 * time.Millisecond,
		},
		Persistence: PersistenceConfig{
			Snapshot: SnapshotConfig{
				Enabled:         false,
				IntervalSeconds: 300,
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
func LoadConfig(path string) (*Config, error) {
	cfg := DefaultConfig()

	// If config file doesn't exist, return default config
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return cfg, nil
	}

	data, err := os.ReadFile(path)
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
func (c *Config) Validate() error {
	if c.Server.Shards <= 0 {
		return fmt.Errorf("server.shards must be greater than 0")
	}

	if c.Limits.MaxClients <= 0 {
		return fmt.Errorf("limits.max_clients must be greater than 0")
	}

	if c.Limits.MaxPipeline <= 0 {
		return fmt.Errorf("limits.max_pipeline must be greater than 0")
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

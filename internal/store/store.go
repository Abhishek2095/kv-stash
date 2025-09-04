// Copyright (c) 2024 Abhishek2095
// SPDX-License-Identifier: MIT

package store

import (
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/Abhishek2095/kv-stash/internal/obs"
)

// Store represents the main key-value store
type Store struct {
	config       *Config
	logger       *obs.Logger
	shards       []*Shard
	expiredCount int64
}

// Config represents store configuration
type Config struct {
	Shards         int
	MaxMemoryBytes int64
	EvictionPolicy string
}

// Shard represents a single shard of the store
type Shard struct {
	id     int
	mu     sync.RWMutex
	data   map[string]*Value
	logger *obs.Logger
}

// Value represents a stored value with metadata
type Value struct {
	Data      string
	Type      ValueType
	ExpiresAt *time.Time
	Version   uint64
}

// ValueType represents the type of value
type ValueType int

const (
	StringType ValueType = iota
	IntegerType
)

// New creates a new store instance
func New(config *Config, logger *obs.Logger) (*Store, error) {
	if config.Shards <= 0 {
		return nil, fmt.Errorf("shards must be greater than 0")
	}

	store := &Store{
		config: config,
		logger: logger,
		shards: make([]*Shard, config.Shards),
	}

	// Initialize shards
	for i := 0; i < config.Shards; i++ {
		store.shards[i] = &Shard{
			id:     i,
			data:   make(map[string]*Value),
			logger: logger.WithFields("shard", i),
		}
	}

	logger.Info("Store initialized", "shards", config.Shards)
	return store, nil
}

// getShard returns the shard for a given key
func (s *Store) getShard(key string) *Shard {
	hash := fnv1aHash(key)
	return s.shards[hash%uint32(len(s.shards))]
}

// Get retrieves a value by key
func (s *Store) Get(key string) (string, bool) {
	shard := s.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	value, exists := shard.data[key]
	if !exists {
		return "", false
	}

	// Check if value has expired
	if value.ExpiresAt != nil && time.Now().After(*value.ExpiresAt) {
		// Remove expired key (lazy expiration)
		delete(shard.data, key)
		atomic.AddInt64(&s.expiredCount, 1)
		return "", false
	}

	return value.Data, true
}

// Set stores a value with optional expiration
func (s *Store) Set(key, value string, expiration *time.Duration) {
	shard := s.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	val := &Value{
		Data:    value,
		Type:    StringType,
		Version: uint64(time.Now().UnixNano()),
	}

	if expiration != nil {
		expiresAt := time.Now().Add(*expiration)
		val.ExpiresAt = &expiresAt
	}

	shard.data[key] = val
}

// Delete removes a key
func (s *Store) Delete(key string) bool {
	shard := s.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	_, exists := shard.data[key]
	if exists {
		delete(shard.data, key)
	}

	return exists
}

// Exists checks if a key exists
func (s *Store) Exists(key string) bool {
	shard := s.getShard(key)
	shard.mu.RLock()
	defer shard.mu.RUnlock()

	value, exists := shard.data[key]
	if !exists {
		return false
	}

	// Check if value has expired
	if value.ExpiresAt != nil && time.Now().After(*value.ExpiresAt) {
		return false
	}

	return true
}

// Expire sets an expiration time for a key
func (s *Store) Expire(key string, duration time.Duration) bool {
	shard := s.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	value, exists := shard.data[key]
	if !exists {
		return false
	}

	expiresAt := time.Now().Add(duration)
	value.ExpiresAt = &expiresAt
	return true
}

// TTL returns the time to live for a key
func (s *Store) TTL(key string) int64 {
	shard := s.getShard(key)
	shard.mu.Lock()
	defer shard.mu.Unlock()

	value, exists := shard.data[key]
	if !exists {
		return -2 // key does not exist
	}

	if value.ExpiresAt == nil {
		return -1 // key exists but has no expiration
	}

	ttl := time.Until(*value.ExpiresAt)
	if ttl <= 0 {
		// Clean up expired key
		delete(shard.data, key)
		atomic.AddInt64(&s.expiredCount, 1)
		return -2 // key has expired
	}

	// Return TTL in seconds, but ensure it's at least 1 if positive
	ttlSeconds := int64(ttl.Seconds())
	if ttlSeconds == 0 && ttl > 0 {
		return 1 // Round up sub-second TTLs to 1 second
	}

	return ttlSeconds
}

// DBSize returns the total number of keys
func (s *Store) DBSize() int64 {
	var total int64
	for _, shard := range s.shards {
		shard.mu.RLock()
		total += int64(len(shard.data))
		shard.mu.RUnlock()
	}
	return total
}

// fnv1aHash implements FNV-1a hash algorithm
func fnv1aHash(key string) uint32 {
	const (
		fnvPrime = 16777619
		fnvBasis = 2166136261
	)

	hash := uint32(fnvBasis)
	for i := 0; i < len(key); i++ {
		hash ^= uint32(key[i])
		hash *= fnvPrime
	}
	return hash
}

// GetExpiredKeysCount returns the total number of expired keys
func (s *Store) GetExpiredKeysCount() int64 {
	return atomic.LoadInt64(&s.expiredCount)
}

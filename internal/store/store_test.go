package store_test

import (
	"testing"
	"time"

	"github.com/Abhishek2095/kv-stash/internal/obs"
	"github.com/Abhishek2095/kv-stash/internal/store"
)

func TestStore_BasicOperations(t *testing.T) {
	logger := obs.NewLogger(true)
	config := &store.Config{
		Shards:         4,
		MaxMemoryBytes: 0,
		EvictionPolicy: "noeviction",
	}

	s, err := store.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Test SET and GET
	s.Set("key1", "value1", nil)
	value, exists := s.Get("key1")
	if !exists || value != "value1" {
		t.Errorf("Expected value1, got %v (exists: %v)", value, exists)
	}

	// Test non-existent key
	_, exists = s.Get("nonexistent")
	if exists {
		t.Errorf("Expected key to not exist")
	}

	// Test DELETE
	deleted := s.Delete("key1")
	if !deleted {
		t.Errorf("Expected key to be deleted")
	}

	_, exists = s.Get("key1")
	if exists {
		t.Errorf("Expected key to not exist after deletion")
	}

	// Test EXISTS
	s.Set("key2", "value2", nil)
	if !s.Exists("key2") {
		t.Errorf("Expected key2 to exist")
	}

	if s.Exists("nonexistent") {
		t.Errorf("Expected nonexistent key to not exist")
	}
}

func TestStore_TTL(t *testing.T) {
	logger := obs.NewLogger(true)
	config := &store.Config{
		Shards:         4,
		MaxMemoryBytes: 0,
		EvictionPolicy: "noeviction",
	}

	s, err := store.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Test SET with expiration
	expiration := 100 * time.Millisecond
	s.Set("expiring_key", "value", &expiration)

	// Should exist immediately
	if !s.Exists("expiring_key") {
		t.Errorf("Expected key to exist immediately after setting")
	}

	// Check TTL
	ttl := s.TTL("expiring_key")
	if ttl <= 0 || ttl > 1 {
		t.Errorf("Expected TTL to be positive and less than 1 second, got %d", ttl)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not exist after expiration
	if s.Exists("expiring_key") {
		t.Errorf("Expected key to not exist after expiration")
	}

	// TTL should be -2 for non-existent key
	ttl = s.TTL("expiring_key")
	if ttl != -2 {
		t.Errorf("Expected TTL to be -2 for expired key, got %d", ttl)
	}
}

func TestStore_EXPIRE(t *testing.T) {
	logger := obs.NewLogger(true)
	config := &store.Config{
		Shards:         4,
		MaxMemoryBytes: 0,
		EvictionPolicy: "noeviction",
	}

	s, err := store.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Set key without expiration
	s.Set("key", "value", nil)

	// TTL should be -1 (no expiration)
	ttl := s.TTL("key")
	if ttl != -1 {
		t.Errorf("Expected TTL to be -1 for key without expiration, got %d", ttl)
	}

	// Set expiration
	duration := 100 * time.Millisecond
	success := s.Expire("key", duration)
	if !success {
		t.Errorf("Expected EXPIRE to succeed")
	}

	// TTL should now be positive
	ttl = s.TTL("key")
	if ttl <= 0 {
		t.Errorf("Expected TTL to be positive after EXPIRE, got %d", ttl)
	}

	// Try to expire non-existent key
	success = s.Expire("nonexistent", duration)
	if success {
		t.Errorf("Expected EXPIRE to fail for non-existent key")
	}
}

func TestStore_DBSize(t *testing.T) {
	logger := obs.NewLogger(true)
	config := &store.Config{
		Shards:         4,
		MaxMemoryBytes: 0,
		EvictionPolicy: "noeviction",
	}

	s, err := store.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Initially empty
	if s.DBSize() != 0 {
		t.Errorf("Expected DBSize to be 0 initially, got %d", s.DBSize())
	}

	// Add some keys
	s.Set("key1", "value1", nil)
	s.Set("key2", "value2", nil)
	s.Set("key3", "value3", nil)

	if s.DBSize() != 3 {
		t.Errorf("Expected DBSize to be 3, got %d", s.DBSize())
	}

	// Delete a key
	s.Delete("key2")

	if s.DBSize() != 2 {
		t.Errorf("Expected DBSize to be 2 after deletion, got %d", s.DBSize())
	}
}

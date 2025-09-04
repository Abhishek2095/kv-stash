package store_test

import (
	"fmt"
	"testing"
	"time"

	"github.com/Abhishek2095/kv-stash/internal/obs"
	"github.com/Abhishek2095/kv-stash/internal/store"
)

func TestStore_BasicOperations(t *testing.T) {
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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
	t.Parallel()

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

func TestStore_Configuration(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)

	// Test with invalid shard count
	invalidConfig := &store.Config{
		Shards:         0,
		MaxMemoryBytes: 0,
		EvictionPolicy: "noeviction",
	}

	_, err := store.New(invalidConfig, logger)
	if err == nil {
		t.Errorf("Expected error with zero shards")
	}

	expectedErrMsg := "shards must be greater than 0"
	if err.Error() != expectedErrMsg {
		t.Errorf("Expected error message %q, got %q", expectedErrMsg, err.Error())
	}

	// Test with negative shard count
	invalidConfig.Shards = -1
	_, err = store.New(invalidConfig, logger)
	if err == nil {
		t.Errorf("Expected error with negative shards")
	}
}

func TestStore_ConcurrentOperations(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)
	config := &store.Config{
		Shards:         8,
		MaxMemoryBytes: 0,
		EvictionPolicy: "noeviction",
	}

	s, err := store.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Test concurrent SET operations
	const numGoroutines = 100
	const numOpsPerGoroutine = 10

	done := make(chan bool, numGoroutines)

	for i := 0; i < numGoroutines; i++ {
		go func(goroutineID int) {
			defer func() { done <- true }()

			for j := 0; j < numOpsPerGoroutine; j++ {
				key := fmt.Sprintf("key_%d_%d", goroutineID, j)
				value := fmt.Sprintf("value_%d_%d", goroutineID, j)
				s.Set(key, value, nil)

				// Verify the value was set
				retrievedValue, exists := s.Get(key)
				if !exists || retrievedValue != value {
					t.Errorf("Concurrent SET/GET failed for key %s", key)
				}
			}
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < numGoroutines; i++ {
		<-done
	}

	// Verify final DB size
	expectedSize := int64(numGoroutines * numOpsPerGoroutine)
	if s.DBSize() != expectedSize {
		t.Errorf("Expected DBSize to be %d, got %d", expectedSize, s.DBSize())
	}
}

func TestStore_ExpiredKeysBehavior(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)
	config := &store.Config{
		Shards:         4,
		MaxMemoryBytes: 0,
		EvictionPolicy: "noeviction",
	}

	s, err := store.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Test expired key counter
	initialExpiredCount := s.GetExpiredKeysCount()
	if initialExpiredCount != 0 {
		t.Errorf("Expected initial expired keys count to be 0, got %d", initialExpiredCount)
	}

	// Set key with very short expiration
	shortExpiration := 1 * time.Millisecond
	s.Set("short_lived", "value", &shortExpiration)

	// Wait for expiration
	time.Sleep(10 * time.Millisecond)

	// Try to get the key (should trigger lazy expiration)
	_, exists := s.Get("short_lived")
	if exists {
		t.Errorf("Expected key to be expired")
	}

	// Check that expired counter was incremented
	expiredCount := s.GetExpiredKeysCount()
	if expiredCount <= initialExpiredCount {
		t.Errorf("Expected expired keys count to increase")
	}
}

func TestStore_TTLEdgeCases(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)
	config := &store.Config{
		Shards:         4,
		MaxMemoryBytes: 0,
		EvictionPolicy: "noeviction",
	}

	s, err := store.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Test TTL for non-existent key
	ttl := s.TTL("nonexistent")
	if ttl != -2 {
		t.Errorf("Expected TTL to be -2 for non-existent key, got %d", ttl)
	}

	// Test TTL for key without expiration
	s.Set("permanent", "value", nil)
	ttl = s.TTL("permanent")
	if ttl != -1 {
		t.Errorf("Expected TTL to be -1 for permanent key, got %d", ttl)
	}

	// Test TTL for key with very short expiration (sub-second)
	shortExpiration := 500 * time.Millisecond
	s.Set("short_ttl", "value", &shortExpiration)
	ttl = s.TTL("short_ttl")
	if ttl != 1 {
		t.Errorf("Expected TTL to be rounded up to 1 second, got %d", ttl)
	}

	// Test TTL for expired key
	veryShortExpiration := 1 * time.Millisecond
	s.Set("expired", "value", &veryShortExpiration)
	time.Sleep(10 * time.Millisecond)

	ttl = s.TTL("expired")
	if ttl != -2 {
		t.Errorf("Expected TTL to be -2 for expired key, got %d", ttl)
	}
}

func TestStore_DeleteOperations(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)
	config := &store.Config{
		Shards:         4,
		MaxMemoryBytes: 0,
		EvictionPolicy: "noeviction",
	}

	s, err := store.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Test deleting non-existent key
	deleted := s.Delete("nonexistent")
	if deleted {
		t.Errorf("Expected Delete to return false for non-existent key")
	}

	// Test deleting existing key
	s.Set("key1", "value1", nil)
	deleted = s.Delete("key1")
	if !deleted {
		t.Errorf("Expected Delete to return true for existing key")
	}

	// Verify key is actually deleted
	_, exists := s.Get("key1")
	if exists {
		t.Errorf("Expected key to not exist after deletion")
	}

	// Test deleting already deleted key
	deleted = s.Delete("key1")
	if deleted {
		t.Errorf("Expected Delete to return false for already deleted key")
	}
}

func TestStore_ExistsOperations(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)
	config := &store.Config{
		Shards:         4,
		MaxMemoryBytes: 0,
		EvictionPolicy: "noeviction",
	}

	s, err := store.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Test Exists for non-existent key
	if s.Exists("nonexistent") {
		t.Errorf("Expected Exists to return false for non-existent key")
	}

	// Test Exists for existing key
	s.Set("key1", "value1", nil)
	if !s.Exists("key1") {
		t.Errorf("Expected Exists to return true for existing key")
	}

	// Test Exists for expired key
	expiration := 1 * time.Millisecond
	s.Set("expiring", "value", &expiration)
	time.Sleep(10 * time.Millisecond)

	if s.Exists("expiring") {
		t.Errorf("Expected Exists to return false for expired key")
	}
}

func TestStore_ShardDistribution(t *testing.T) {
	t.Parallel()

	logger := obs.NewLogger(false)
	config := &store.Config{
		Shards:         4,
		MaxMemoryBytes: 0,
		EvictionPolicy: "noeviction",
	}

	s, err := store.New(config, logger)
	if err != nil {
		t.Fatalf("Failed to create store: %v", err)
	}

	// Set multiple keys and verify they can all be retrieved
	// This tests that sharding works correctly
	keys := []string{
		"shard_test_1", "shard_test_2", "shard_test_3", "shard_test_4",
		"another_key", "yet_another", "final_key", "last_one",
	}

	for i, key := range keys {
		value := fmt.Sprintf("value_%d", i)
		s.Set(key, value, nil)
	}

	// Verify all keys can be retrieved
	for i, key := range keys {
		expectedValue := fmt.Sprintf("value_%d", i)
		value, exists := s.Get(key)
		if !exists {
			t.Errorf("Key %s was not found", key)
		}
		if value != expectedValue {
			t.Errorf("Key %s has value %s, expected %s", key, value, expectedValue)
		}
	}

	// Verify DBSize reflects all keys
	if s.DBSize() != int64(len(keys)) {
		t.Errorf("Expected DBSize to be %d, got %d", len(keys), s.DBSize())
	}
}

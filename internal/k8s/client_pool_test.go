package k8s

import (
	"testing"
	"time"
)

func TestClientPoolEvictsLRU(t *testing.T) {
	pool := NewClientPool(10*time.Minute, 2)

	// Manually insert two entries to test eviction logic
	pool.mu.Lock()
	pool.clients["cluster-a"] = &SpokeClient{CreatedAt: time.Now()}
	pool.accesses["cluster-a"] = time.Now().Add(-5 * time.Minute)
	pool.clients["cluster-b"] = &SpokeClient{CreatedAt: time.Now()}
	pool.accesses["cluster-b"] = time.Now()
	pool.mu.Unlock()

	// Pool is at capacity (2). Eviction should remove cluster-a (oldest access)
	pool.mu.Lock()
	pool.evictLRU()
	pool.mu.Unlock()

	pool.mu.Lock()
	defer pool.mu.Unlock()

	if _, ok := pool.clients["cluster-a"]; ok {
		t.Error("expected cluster-a to be evicted")
	}
	if _, ok := pool.clients["cluster-b"]; !ok {
		t.Error("expected cluster-b to remain")
	}
}

func TestClientPoolTTLExpiration(t *testing.T) {
	pool := NewClientPool(1*time.Millisecond, 10)

	// Insert an expired client
	pool.mu.Lock()
	pool.clients["expired"] = &SpokeClient{CreatedAt: time.Now().Add(-1 * time.Hour)}
	pool.accesses["expired"] = time.Now()
	pool.mu.Unlock()

	// GetOrCreate with invalid kubeconfig should fail, but the expired entry should be cleaned up
	_, err := pool.GetOrCreate("expired", []byte("invalid"))
	if err == nil {
		t.Error("expected error for invalid kubeconfig")
	}

	pool.mu.Lock()
	_, exists := pool.clients["expired"]
	pool.mu.Unlock()

	if exists {
		t.Error("expected expired client to be removed")
	}
}

func TestNewClientPool(t *testing.T) {
	pool := NewClientPool(5*time.Minute, 10)
	if pool == nil {
		t.Fatal("expected non-nil pool")
	}
	if pool.ttl != 5*time.Minute {
		t.Errorf("expected TTL 5m, got %v", pool.ttl)
	}
	if pool.maxSize != 10 {
		t.Errorf("expected max size 10, got %d", pool.maxSize)
	}
	if len(pool.clients) != 0 {
		t.Errorf("expected empty clients map, got %d entries", len(pool.clients))
	}
}

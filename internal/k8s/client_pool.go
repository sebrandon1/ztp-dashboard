package k8s

import (
	"fmt"
	"log/slog"
	"sync"
	"time"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

// SpokeClient holds a K8s client for a spoke cluster with expiration tracking.
type SpokeClient struct {
	Clientset kubernetes.Interface
	Dynamic   dynamic.Interface
	CreatedAt time.Time
}

// ClientPool manages a pool of spoke K8s clients with LRU eviction and TTL.
type ClientPool struct {
	mu       sync.Mutex
	clients  map[string]*SpokeClient
	ttl      time.Duration
	maxSize  int
	accesses map[string]time.Time
}

// NewClientPool creates a new client pool with the given TTL and max size.
func NewClientPool(ttl time.Duration, maxSize int) *ClientPool {
	return &ClientPool{
		clients:  make(map[string]*SpokeClient),
		ttl:      ttl,
		maxSize:  maxSize,
		accesses: make(map[string]time.Time),
	}
}

// GetOrCreate returns a cached spoke client or creates one from the kubeconfig bytes.
func (p *ClientPool) GetOrCreate(clusterName string, kubeconfigBytes []byte) (*SpokeClient, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	// Check for cached client that hasn't expired
	if client, ok := p.clients[clusterName]; ok {
		if time.Since(client.CreatedAt) < p.ttl {
			p.accesses[clusterName] = time.Now()
			return client, nil
		}
		// Expired — remove it
		delete(p.clients, clusterName)
		delete(p.accesses, clusterName)
		slog.Debug("spoke client expired", "cluster", clusterName)
	}

	// Evict LRU if at capacity
	if len(p.clients) >= p.maxSize {
		p.evictLRU()
	}

	// Build new client from kubeconfig
	restConfig, err := clientcmd.RESTConfigFromKubeConfig(kubeconfigBytes)
	if err != nil {
		return nil, fmt.Errorf("building rest config for spoke %s: %w", clusterName, err)
	}

	// Set reasonable timeouts for spoke connections
	restConfig.Timeout = 15 * time.Second

	clientset, err := kubernetes.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating clientset for spoke %s: %w", clusterName, err)
	}

	dynClient, err := dynamic.NewForConfig(restConfig)
	if err != nil {
		return nil, fmt.Errorf("creating dynamic client for spoke %s: %w", clusterName, err)
	}

	client := &SpokeClient{
		Clientset: clientset,
		Dynamic:   dynClient,
		CreatedAt: time.Now(),
	}

	p.clients[clusterName] = client
	p.accesses[clusterName] = time.Now()
	slog.Debug("created spoke client", "cluster", clusterName)

	return client, nil
}

// evictLRU removes the least recently accessed client. Must be called with lock held.
func (p *ClientPool) evictLRU() {
	var oldestName string
	var oldestTime time.Time

	for name, accessTime := range p.accesses {
		if oldestName == "" || accessTime.Before(oldestTime) {
			oldestName = name
			oldestTime = accessTime
		}
	}

	if oldestName != "" {
		delete(p.clients, oldestName)
		delete(p.accesses, oldestName)
		slog.Debug("evicted spoke client", "cluster", oldestName)
	}
}

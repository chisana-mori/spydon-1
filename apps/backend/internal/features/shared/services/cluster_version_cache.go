package services

import (
	"strings"
	"sync"
	"time"

	"k8s.io/client-go/kubernetes"
)

type clusterVersionEntry struct {
	useBeta  bool
	expireAt time.Time
}

type ClusterVersionCache struct {
	ttl   time.Duration
	mu    sync.RWMutex
	cache map[string]clusterVersionEntry
}

func NewClusterVersionCache(ttl time.Duration) *ClusterVersionCache {
	if ttl <= 0 {
		ttl = 5 * time.Minute
	}
	return &ClusterVersionCache{
		ttl:   ttl,
		cache: make(map[string]clusterVersionEntry),
	}
}

func (c *ClusterVersionCache) ShouldUsePolicyV1beta1(clusterName string, client kubernetes.Interface) bool {
	if client == nil || clusterName == "" {
		return false
	}

	now := time.Now()
	c.mu.RLock()
	entry, ok := c.cache[clusterName]
	if ok && now.Before(entry.expireAt) {
		c.mu.RUnlock()
		return entry.useBeta
	}
	c.mu.RUnlock()

	useBeta := false
	if sv, err := client.Discovery().ServerVersion(); err == nil {
		gitVersion := strings.ToLower(sv.GitVersion)
		useBeta = strings.HasPrefix(gitVersion, "v1.19") || strings.HasPrefix(gitVersion, "v1.20")
	} else if ok {
		return entry.useBeta
	}

	c.mu.Lock()
	c.cache[clusterName] = clusterVersionEntry{useBeta: useBeta, expireAt: now.Add(c.ttl)}
	c.mu.Unlock()
	return useBeta
}

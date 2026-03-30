package proxy

import (
	"context"
	"maps"
	"sync"

	"github.com/bobbyhouse/plugins/gateway/internal/profile"
)

// Cache is an in-memory cache of Sessions keyed by profile OCI reference.
// It also accumulates user-supplied config values per profile so callers
// don't need to re-supply them on every invoke.
type Cache struct {
	mu         sync.Mutex
	sessions   map[string]*Session
	userConfig map[string]map[string]string // ref → accumulated key/value pairs
}

// NewCache returns an empty Cache.
func NewCache() *Cache {
	return &Cache{
		sessions:   make(map[string]*Session),
		userConfig: make(map[string]map[string]string),
	}
}

// GetOrLoad returns the cached Session for ref, or loads a new one.
// config values are merged into the accumulated config for ref before loading.
// Returns *profile.MissingConfigError if any ${KEY} placeholders remain unresolved.
func (c *Cache) GetOrLoad(ctx context.Context, ref string, config map[string]string) (*Session, error) {
	c.mu.Lock()
	// Merge caller-supplied config into the accumulated map for this ref.
	if c.userConfig[ref] == nil {
		c.userConfig[ref] = make(map[string]string)
	}
	maps.Copy(c.userConfig[ref], config)
	// Copy merged config before releasing the lock so it can't be mutated
	// by a concurrent call while we're loading.
	merged := make(map[string]string, len(c.userConfig[ref]))
	maps.Copy(merged, c.userConfig[ref])
	if s, ok := c.sessions[ref]; ok {
		c.mu.Unlock()
		return s, nil
	}
	c.mu.Unlock()

	p, err := profile.Load(ctx, ref, merged)
	if err != nil {
		return nil, err
	}
	s, err := NewSession(ctx, p)
	if err != nil {
		return nil, err
	}

	c.mu.Lock()
	// Check again in case another goroutine raced us.
	if existing, ok := c.sessions[ref]; ok {
		c.mu.Unlock()
		s.Close()
		return existing, nil
	}
	c.sessions[ref] = s
	c.mu.Unlock()
	return s, nil
}

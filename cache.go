// Package cache is togo's default in-memory cache provider. Blank-import (or
// `togo install togo-framework/cache`) to register it with the kernel.
package cache

import (
	"sync"
	"time"

	"github.com/togo-framework/togo"
)

func init() {
	togo.RegisterProviderFunc("cache", togo.PriorityService, func(k *togo.Kernel) error {
		k.Cache = NewMemory()
		return nil
	})
}

type entry struct {
	value   any
	expires time.Time
}

type memory struct {
	mu    sync.RWMutex
	items map[string]entry
}

// NewMemory returns an in-memory cache.
func NewMemory() togo.Cache { return &memory{items: map[string]entry{}} }

func (m *memory) Get(key string) (any, bool) {
	m.mu.RLock()
	e, ok := m.items[key]
	m.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if !e.expires.IsZero() && time.Now().After(e.expires) {
		m.Delete(key)
		return nil, false
	}
	return e.value, true
}

func (m *memory) Set(key string, value any, ttl time.Duration) {
	var exp time.Time
	if ttl > 0 {
		exp = time.Now().Add(ttl)
	}
	m.mu.Lock()
	m.items[key] = entry{value: value, expires: exp}
	m.mu.Unlock()
}

func (m *memory) Delete(key string) {
	m.mu.Lock()
	delete(m.items, key)
	m.mu.Unlock()
}

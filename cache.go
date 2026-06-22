// Package cache is togo's cache provider with pluggable drivers: memory (default),
// file, and database are built in; redis and others register via RegisterDriver.
// Select with CACHE_DRIVER. Blank-import (or `togo install togo-framework/cache`).
package cache

import (
	"fmt"
	"os"
	"sync"
	"time"

	"github.com/togo-framework/togo"
)

// DriverFactory builds a cache from the kernel.
type DriverFactory func(k *togo.Kernel) (togo.Cache, error)

var (
	regMu   sync.RWMutex
	drivers = map[string]DriverFactory{}
)

// RegisterDriver registers a cache driver by name (call from a plugin init()).
func RegisterDriver(name string, f DriverFactory) {
	regMu.Lock()
	drivers[name] = f
	regMu.Unlock()
}

func init() {
	RegisterDriver("memory", func(*togo.Kernel) (togo.Cache, error) { return NewMemory(), nil })
	RegisterDriver("file", func(*togo.Kernel) (togo.Cache, error) { return newFileCache(), nil })
	RegisterDriver("database", func(k *togo.Kernel) (togo.Cache, error) { return newDBCache(k) })

	togo.RegisterProviderFunc("cache", togo.PriorityService, func(k *togo.Kernel) error {
		name := os.Getenv("CACHE_DRIVER")
		if name == "" {
			name = "memory"
		}
		regMu.RLock()
		f, ok := drivers[name]
		regMu.RUnlock()
		if !ok {
			return fmt.Errorf("cache: unknown driver %q (install its plugin?)", name)
		}
		c, err := f(k)
		if err != nil {
			return err
		}
		k.Cache = c
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

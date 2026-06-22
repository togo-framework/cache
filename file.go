package cache

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/togo-framework/togo"
)

// fileCache persists JSON-serialisable values as files under CACHE_DIR.
type fileCache struct {
	dir string
	mu  sync.Mutex
}

func newFileCache() togo.Cache {
	dir := os.Getenv("CACHE_DIR")
	if dir == "" {
		dir = "storage/cache"
	}
	_ = os.MkdirAll(dir, 0o700) //#nosec G703 -- dir is operator config (CACHE_DIR), not user input
	return &fileCache{dir: dir}
}

type fileEntry struct {
	Value   json.RawMessage `json:"value"`
	Expires int64           `json:"expires"` // unix; 0 = never
}

func (f *fileCache) path(key string) string { return filepath.Join(f.dir, "c_"+hashKey(key)+".json") }

func (f *fileCache) Get(key string) (any, bool) {
	f.mu.Lock()
	defer f.mu.Unlock()
	b, err := os.ReadFile(f.path(key)) //#nosec G304 -- path derived from a hashed key under CACHE_DIR
	if err != nil {
		return nil, false
	}
	var e fileEntry
	if json.Unmarshal(b, &e) != nil {
		return nil, false
	}
	if e.Expires != 0 && time.Now().Unix() > e.Expires {
		_ = os.Remove(f.path(key))
		return nil, false
	}
	var v any
	_ = json.Unmarshal(e.Value, &v)
	return v, true
}

func (f *fileCache) Set(key string, value any, ttl time.Duration) {
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	e := fileEntry{Value: raw}
	if ttl > 0 {
		e.Expires = time.Now().Add(ttl).Unix()
	}
	data, _ := json.Marshal(e)
	f.mu.Lock()
	_ = os.WriteFile(f.path(key), data, 0o600)
	f.mu.Unlock()
}

func (f *fileCache) Delete(key string) {
	f.mu.Lock()
	_ = os.Remove(f.path(key))
	f.mu.Unlock()
}

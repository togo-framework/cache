package cache

import (
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"time"

	"github.com/togo-framework/togo"
)

// dbCache stores JSON-serialisable values in a cache_entries table.
type dbCache struct {
	k  *togo.Kernel
	db *sql.DB
}

func newDBCache(k *togo.Kernel) (togo.Cache, error) {
	db, err := k.SQL(context.Background())
	if err != nil {
		return nil, err
	}
	_, err = db.ExecContext(context.Background(), `CREATE TABLE IF NOT EXISTS cache_entries (
		k text PRIMARY KEY,
		value text NOT NULL,
		expires_at integer NOT NULL DEFAULT 0
	)`)
	if err != nil {
		return nil, err
	}
	return &dbCache{k: k, db: db}, nil
}

func (c *dbCache) ph(n int) string { return c.k.Dialect().Placeholder(n) }

func (c *dbCache) Get(key string) (any, bool) {
	var raw string
	var exp int64
	//#nosec G202 -- dialect placeholder only; value parameterized
	row := c.db.QueryRowContext(context.Background(), "SELECT value, expires_at FROM cache_entries WHERE k = "+c.ph(1), hashKey(key))
	if row.Scan(&raw, &exp) != nil {
		return nil, false
	}
	if exp != 0 && time.Now().Unix() > exp {
		c.Delete(key)
		return nil, false
	}
	var v any
	_ = json.Unmarshal([]byte(raw), &v)
	return v, true
}

func (c *dbCache) Set(key string, value any, ttl time.Duration) {
	raw, err := json.Marshal(value)
	if err != nil {
		return
	}
	var exp int64
	if ttl > 0 {
		exp = time.Now().Add(ttl).Unix()
	}
	p := c.ph
	//#nosec G202 -- dialect placeholders only; values parameterized
	q := "INSERT INTO cache_entries (k, value, expires_at) VALUES (" + p(1) + ", " + p(2) + ", " + p(3) +
		") ON CONFLICT (k) DO UPDATE SET value = " + p(2) + ", expires_at = " + p(3)
	_, _ = c.db.ExecContext(context.Background(), q, hashKey(key), string(raw), exp)
}

func (c *dbCache) Delete(key string) {
	//#nosec G202 -- dialect placeholder only; value parameterized
	_, _ = c.db.ExecContext(context.Background(), "DELETE FROM cache_entries WHERE k = "+c.ph(1), hashKey(key))
}

// hashKey normalises arbitrary keys into a safe, fixed-length identifier.
func hashKey(key string) string {
	sum := sha256.Sum256([]byte(key))
	return hex.EncodeToString(sum[:])
}

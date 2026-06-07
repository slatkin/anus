package cache

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	bolt "go.etcd.io/bbolt"

	"github.com/slatkin/anus/pkg/miniflux"
)

var bucket = []byte("articles")

type row struct {
	E         miniflux.FeedEntry `json:"e"`
	FetchedAt time.Time          `json:"fa"`
}

type Cache struct {
	db     *bolt.DB
	expiry time.Duration
}

// DefaultDir returns the default cache directory (os.UserCacheDir/anus).
func DefaultDir() (string, error) {
	base, err := os.UserCacheDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "anus"), nil
}

func Open(dir string, expiryDays int) (*Cache, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, err
	}
	db, err := bolt.Open(filepath.Join(dir, "articles.db"), 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		return nil, err
	}
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	}); err != nil {
		db.Close()
		return nil, err
	}
	return &Cache{db: db, expiry: time.Duration(expiryDays) * 24 * time.Hour}, nil
}

func (c *Cache) Close() error { return c.db.Close() }

// Put stores entries. For entries already in cache, FetchedAt is preserved so
// the 30-day clock starts from when the article was first seen, not last seen.
func (c *Cache) Put(entries []miniflux.FeedEntry) error {
	now := time.Now()
	return c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		for _, e := range entries {
			key := key(e.ID)
			fetchedAt := now
			if raw := b.Get(key); raw != nil {
				var r row
				if json.Unmarshal(raw, &r) == nil {
					fetchedAt = r.FetchedAt
				}
			}
			data, err := json.Marshal(row{E: e, FetchedAt: fetchedAt})
			if err != nil {
				return err
			}
			if err := b.Put(key, data); err != nil {
				return err
			}
		}
		return nil
	})
}

// Update applies fn to a single cached entry (e.g. to sync read/star state).
func (c *Cache) Update(id int, fn func(*miniflux.FeedEntry)) error {
	return c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		k := key(id)
		raw := b.Get(k)
		if raw == nil {
			return nil
		}
		var r row
		if err := json.Unmarshal(raw, &r); err != nil {
			return err
		}
		fn(&r.E)
		data, err := json.Marshal(r)
		if err != nil {
			return err
		}
		return b.Put(k, data)
	})
}

// All returns every non-expired entry.
func (c *Cache) All() ([]miniflux.FeedEntry, error) {
	cutoff := time.Now().Add(-c.expiry)
	var out []miniflux.FeedEntry
	err := c.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket(bucket).ForEach(func(_, v []byte) error {
			var r row
			if json.Unmarshal(v, &r) != nil || r.FetchedAt.Before(cutoff) {
				return nil
			}
			r.E.FetchedAt = r.FetchedAt
			out = append(out, r.E)
			return nil
		})
	})
	return out, err
}

// Purge deletes entries older than the expiry window.
func (c *Cache) Purge() error {
	cutoff := time.Now().Add(-c.expiry)
	return c.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket(bucket)
		var stale [][]byte
		b.ForEach(func(k, v []byte) error { //nolint
			var r row
			if json.Unmarshal(v, &r) != nil || r.FetchedAt.Before(cutoff) {
				stale = append(stale, append([]byte{}, k...))
			}
			return nil
		})
		for _, k := range stale {
			if err := b.Delete(k); err != nil {
				return err
			}
		}
		return nil
	})
}

// Clear deletes all entries from the cache.
func (c *Cache) Clear() error {
	return c.db.Update(func(tx *bolt.Tx) error {
		if err := tx.DeleteBucket(bucket); err != nil {
			return err
		}
		_, err := tx.CreateBucket(bucket)
		return err
	})
}

func key(id int) []byte { return []byte(fmt.Sprintf("%d", id)) }

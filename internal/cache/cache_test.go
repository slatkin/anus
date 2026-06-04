package cache

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/slatkin/anus/pkg/miniflux"
	bolt "go.etcd.io/bbolt"
)

func openTestCache(t *testing.T, expiryDays int) *Cache {
	t.Helper()
	dir := t.TempDir()
	db, err := bolt.Open(filepath.Join(dir, "test.db"), 0600, &bolt.Options{Timeout: time.Second})
	if err != nil {
		t.Fatal(err)
	}
	if err := db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists(bucket)
		return err
	}); err != nil {
		db.Close()
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	return &Cache{db: db, expiry: time.Duration(expiryDays) * 24 * time.Hour}
}

func entry(id int) miniflux.FeedEntry {
	return miniflux.FeedEntry{ID: id, Title: "entry", Status: miniflux.ReadStatusUnread}
}

func TestKey(t *testing.T) {
	if string(key(42)) != "42" {
		t.Errorf("key(42) = %q, want \"42\"", key(42))
	}
	if string(key(0)) != "0" {
		t.Errorf("key(0) = %q, want \"0\"", key(0))
	}
}

func TestPutAndAll(t *testing.T) {
	c := openTestCache(t, 30)
	entries := []miniflux.FeedEntry{entry(1), entry(2)}
	if err := c.Put(entries); err != nil {
		t.Fatal(err)
	}
	got, err := c.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Errorf("All() returned %d entries, want 2", len(got))
	}
}

func TestPutPreservesFetchedAt(t *testing.T) {
	c := openTestCache(t, 30)

	first := []miniflux.FeedEntry{entry(1)}
	if err := c.Put(first); err != nil {
		t.Fatal(err)
	}
	got1, _ := c.All()
	originalFetchedAt := got1[0].FetchedAt

	time.Sleep(10 * time.Millisecond)

	second := []miniflux.FeedEntry{entry(1)}
	if err := c.Put(second); err != nil {
		t.Fatal(err)
	}
	got2, _ := c.All()
	if !got2[0].FetchedAt.Equal(originalFetchedAt) {
		t.Error("Put() should preserve FetchedAt for existing entries")
	}
}

func TestAllFiltersExpired(t *testing.T) {
	c := openTestCache(t, 30)
	if err := c.Put([]miniflux.FeedEntry{entry(1)}); err != nil {
		t.Fatal(err)
	}

	// Move expiry to 0 days so everything appears expired.
	c.expiry = 0
	got, err := c.All()
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Errorf("All() returned %d entries, want 0 for expired cache", len(got))
	}
}

func TestUpdate(t *testing.T) {
	c := openTestCache(t, 30)
	if err := c.Put([]miniflux.FeedEntry{entry(1)}); err != nil {
		t.Fatal(err)
	}

	if err := c.Update(1, func(e *miniflux.FeedEntry) {
		e.Status = miniflux.ReadStatusRead
	}); err != nil {
		t.Fatal(err)
	}

	got, _ := c.All()
	if len(got) != 1 || got[0].Status != miniflux.ReadStatusRead {
		t.Error("Update() should modify the entry's status")
	}
}

func TestUpdateMissingEntryIsNoop(t *testing.T) {
	c := openTestCache(t, 30)
	// Updating a non-existent entry should not error.
	if err := c.Update(999, func(e *miniflux.FeedEntry) {
		e.Status = miniflux.ReadStatusRead
	}); err != nil {
		t.Errorf("Update() on missing entry returned error: %v", err)
	}
}

func TestPurgeRemovesExpired(t *testing.T) {
	c := openTestCache(t, 30)
	if err := c.Put([]miniflux.FeedEntry{entry(1), entry(2)}); err != nil {
		t.Fatal(err)
	}

	c.expiry = 0
	if err := c.Purge(); err != nil {
		t.Fatal(err)
	}

	// Restore expiry so All() doesn't filter.
	c.expiry = 30 * 24 * time.Hour
	got, _ := c.All()
	if len(got) != 0 {
		t.Errorf("Purge() left %d entries, want 0", len(got))
	}
}

func TestPurgeKeepsFreshEntries(t *testing.T) {
	c := openTestCache(t, 30)
	if err := c.Put([]miniflux.FeedEntry{entry(1), entry(2)}); err != nil {
		t.Fatal(err)
	}
	if err := c.Purge(); err != nil {
		t.Fatal(err)
	}
	got, _ := c.All()
	if len(got) != 2 {
		t.Errorf("Purge() removed fresh entries, got %d want 2", len(got))
	}
}

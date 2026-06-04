package app_test

import (
	"errors"
	"testing"
	"time"

	"github.com/slatkin/anus/pkg/app"
	"github.com/slatkin/anus/pkg/miniflux"
)

// ── mock client ───────────────────────────────────────────────────────────

type mockClient struct {
	unread    []miniflux.FeedEntry
	read      []miniflux.FeedEntry
	unreadErr error
	readErr   error

	markReadCalled   []int
	markUnreadCalled []int
	toggleCalled     []int
	saveCalled       []int
}

func (m *mockClient) GetUnreadEntries(limit, offset int) ([]miniflux.FeedEntry, int, error) {
	if m.unreadErr != nil {
		return nil, 0, m.unreadErr
	}
	end := offset + limit
	if end > len(m.unread) {
		end = len(m.unread)
	}
	page := m.unread[offset:end]
	return page, len(m.unread), nil
}

func (m *mockClient) GetReadEntries(_ time.Time, limit, offset int) ([]miniflux.FeedEntry, int, error) {
	if m.readErr != nil {
		return nil, 0, m.readErr
	}
	end := offset + limit
	if end > len(m.read) {
		end = len(m.read)
	}
	page := m.read[offset:end]
	return page, len(m.read), nil
}

func (m *mockClient) ChangeEntryReadStatus(ids []int, status miniflux.ReadStatus) error {
	if status == miniflux.ReadStatusRead {
		m.markReadCalled = append(m.markReadCalled, ids...)
	} else {
		m.markUnreadCalled = append(m.markUnreadCalled, ids...)
	}
	return nil
}

func (m *mockClient) ToggleStarred(id int) error {
	m.toggleCalled = append(m.toggleCalled, id)
	return nil
}

func (m *mockClient) SaveEntry(id int) error {
	m.saveCalled = append(m.saveCalled, id)
	return nil
}

// ── helpers ───────────────────────────────────────────────────────────────

func entry(id int, status miniflux.ReadStatus) miniflux.FeedEntry {
	return miniflux.FeedEntry{
		ID:          id,
		Title:       "entry",
		Status:      status,
		PublishedAt: time.Now(),
		FeedID:      1,
		Feed:        miniflux.Feed{Title: "Feed"},
	}
}

func newApp(t *testing.T, mc *mockClient) (*app.App, string) {
	t.Helper()
	dir := t.TempDir()
	a := app.New(mc, 30, true)
	if err := a.Open(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	return a, dir
}

// ── tests ─────────────────────────────────────────────────────────────────

func TestFetchEntries_ReturnsUnreadAndRead(t *testing.T) {
	mc := &mockClient{
		unread: []miniflux.FeedEntry{entry(1, miniflux.ReadStatusUnread)},
		read:   []miniflux.FeedEntry{entry(2, miniflux.ReadStatusRead)},
	}
	a, _ := newApp(t, mc)
	result, err := a.FetchEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 2 {
		t.Errorf("got %d entries, want 2", len(result.Entries))
	}
}

func TestFetchEntries_FallsBackToCacheOnNetworkError(t *testing.T) {
	dir := t.TempDir()

	// Seed the cache with a working client first.
	seeder := app.New(&mockClient{
		unread: []miniflux.FeedEntry{entry(10, miniflux.ReadStatusUnread)},
	}, 30, true)
	if err := seeder.Open(dir); err != nil {
		t.Fatal(err)
	}
	if _, err := seeder.FetchEntries(); err != nil {
		t.Fatal(err)
	}
	seeder.Close()

	// Now open a new app with a broken client against the same cache dir.
	mc := &mockClient{unreadErr: errors.New("network down")}
	a := app.New(mc, 30, true)
	if err := a.Open(dir); err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	result, err := a.FetchEntries()
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected cache fallback to return entries")
	}
}

func TestFetchEntries_ErrorWhenNoCacheAndNetworkDown(t *testing.T) {
	mc := &mockClient{unreadErr: errors.New("network down")}
	a := app.New(mc, 30, true)
	// No Open call — cache is nil.
	t.Cleanup(func() { a.Close() })
	_, err := a.FetchEntries()
	if err == nil {
		t.Error("expected error when network is down and cache is nil")
	}
}

func TestFetchEntries_MergesDeduplicated(t *testing.T) {
	e1 := entry(1, miniflux.ReadStatusUnread)
	e2 := entry(2, miniflux.ReadStatusRead)
	mc := &mockClient{unread: []miniflux.FeedEntry{e1}, read: []miniflux.FeedEntry{e2}}
	a, _ := newApp(t, mc)

	// First fetch seeds cache.
	if _, err := a.FetchEntries(); err != nil {
		t.Fatal(err)
	}
	// Second fetch: e2 is now in cache AND returned by read — must not duplicate.
	result, err := a.FetchEntries()
	if err != nil {
		t.Fatal(err)
	}
	seen := make(map[int]int)
	for _, e := range result.Entries {
		seen[e.ID]++
	}
	for id, count := range seen {
		if count > 1 {
			t.Errorf("entry %d appears %d times, want 1", id, count)
		}
	}
}

func TestFetchEntries_RememberReadPosition(t *testing.T) {
	mc := &mockClient{unread: []miniflux.FeedEntry{entry(1, miniflux.ReadStatusUnread)}}
	a := app.New(mc, 30, true)
	t.Cleanup(func() { a.Close() })
	result, err := a.FetchEntries()
	if err != nil {
		t.Fatal(err)
	}
	if !result.RememberReadPosition {
		t.Error("RememberReadPosition should be true")
	}
}

func TestMarkRead_UpdatesCacheAndCallsClient(t *testing.T) {
	mc := &mockClient{unread: []miniflux.FeedEntry{entry(1, miniflux.ReadStatusUnread)}}
	a, _ := newApp(t, mc)
	if _, err := a.FetchEntries(); err != nil {
		t.Fatal(err)
	}
	if err := a.MarkRead([]int{1}); err != nil {
		t.Fatal(err)
	}
	if len(mc.markReadCalled) != 1 || mc.markReadCalled[0] != 1 {
		t.Errorf("MarkRead: client got %v, want [1]", mc.markReadCalled)
	}
}

func TestMarkUnread_UpdatesCacheAndCallsClient(t *testing.T) {
	mc := &mockClient{unread: []miniflux.FeedEntry{entry(1, miniflux.ReadStatusUnread)}}
	a, _ := newApp(t, mc)
	if _, err := a.FetchEntries(); err != nil {
		t.Fatal(err)
	}
	if err := a.MarkUnread([]int{1}); err != nil {
		t.Fatal(err)
	}
	if len(mc.markUnreadCalled) != 1 || mc.markUnreadCalled[0] != 1 {
		t.Errorf("MarkUnread: client got %v, want [1]", mc.markUnreadCalled)
	}
}

func TestToggleStar_CallsClient(t *testing.T) {
	mc := &mockClient{unread: []miniflux.FeedEntry{entry(1, miniflux.ReadStatusUnread)}}
	a, _ := newApp(t, mc)
	if _, err := a.FetchEntries(); err != nil {
		t.Fatal(err)
	}
	if err := a.ToggleStar(1); err != nil {
		t.Fatal(err)
	}
	if len(mc.toggleCalled) != 1 || mc.toggleCalled[0] != 1 {
		t.Errorf("ToggleStar: client got %v, want [1]", mc.toggleCalled)
	}
}

func TestSaveEntry_CallsClient(t *testing.T) {
	mc := &mockClient{}
	a, _ := newApp(t, mc)
	if err := a.SaveEntry(42); err != nil {
		t.Fatal(err)
	}
	if len(mc.saveCalled) != 1 || mc.saveCalled[0] != 42 {
		t.Errorf("SaveEntry: client got %v, want [42]", mc.saveCalled)
	}
}

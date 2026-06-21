package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/slatkin/anus/pkg/app"
	"github.com/slatkin/anus/pkg/config"
	"github.com/slatkin/anus/pkg/miniflux"
)

// ── mock miniflux client ───────────────────────────────────────────────────

type mockClient struct {
	unread      []miniflux.FeedEntry
	read        []miniflux.FeedEntry
	search      []miniflux.FeedEntry
	searchQuery string
}

func (m *mockClient) GetUnreadEntries(limit, offset int) ([]miniflux.FeedEntry, int, error) {
	end := offset + limit
	if end > len(m.unread) {
		end = len(m.unread)
	}
	return m.unread[offset:end], len(m.unread), nil
}

func (m *mockClient) GetReadEntries(_ time.Time, limit, offset int) ([]miniflux.FeedEntry, int, error) {
	end := offset + limit
	if end > len(m.read) {
		end = len(m.read)
	}
	return m.read[offset:end], len(m.read), nil
}

func (m *mockClient) SearchEntries(query string, limit, offset int) ([]miniflux.FeedEntry, int, error) {
	m.searchQuery = query
	end := offset + limit
	if end > len(m.search) {
		end = len(m.search)
	}
	return m.search[offset:end], len(m.search), nil
}

func (m *mockClient) ChangeEntryReadStatus(ids []int, status miniflux.ReadStatus) error { return nil }
func (m *mockClient) ToggleStarred(id int) error                                         { return nil }
func (m *mockClient) SaveEntry(id int) error                                              { return nil }
func (m *mockClient) RefreshAllFeeds() error                                              { return nil }

// ── helpers ───────────────────────────────────────────────────────────────

func makeEntry(id int) miniflux.FeedEntry {
	return miniflux.FeedEntry{
		ID:          id,
		Title:       "test entry",
		Status:      miniflux.ReadStatusUnread,
		PublishedAt: time.Now(),
		FeedID:      1,
		Feed:        miniflux.Feed{Title: "Test Feed"},
	}
}

func newTestServer(t *testing.T, mc *mockClient, cfg config.Config) *httptest.Server {
	t.Helper()
	dir := t.TempDir()
	a := app.New(mc, 30)
	if err := a.Open(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { a.Close() })
	mux := http.NewServeMux()
	registerAPI(mux, a, cfg)
	return httptest.NewServer(mux)
}

func testCfg() config.Config {
	return config.Config{
		ApiKey:    "test-key",
		ServerUrl: "http://example.com",
	}
}

// ── tests ─────────────────────────────────────────────────────────────────

func TestGetCached_200WithEntries(t *testing.T) {
	mc := &mockClient{unread: []miniflux.FeedEntry{makeEntry(1)}}
	srv := newTestServer(t, mc, testCfg())
	defer srv.Close()

	// Seed the cache by calling /api/entries first.
	resp, err := http.Get(srv.URL + "/api/entries")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()

	resp, err = http.Get(srv.URL + "/api/cached")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/cached: got %d, want 200", resp.StatusCode)
	}
	var result app.FetchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) == 0 {
		t.Error("expected at least one entry in cached result")
	}
}

func TestGetEntries_200(t *testing.T) {
	mc := &mockClient{unread: []miniflux.FeedEntry{makeEntry(1), makeEntry(2)}}
	srv := newTestServer(t, mc, testCfg())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/entries")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/entries: got %d, want 200", resp.StatusCode)
	}
	var result app.FetchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 2 {
		t.Errorf("got %d entries, want 2", len(result.Entries))
	}
}

func TestPostMarkRead_204(t *testing.T) {
	mc := &mockClient{unread: []miniflux.FeedEntry{makeEntry(1)}}
	srv := newTestServer(t, mc, testCfg())
	defer srv.Close()

	body := bytes.NewBufferString(`{"ids":[1]}`)
	resp, err := http.Post(srv.URL+"/api/mark-read", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("POST /api/mark-read: got %d, want 204", resp.StatusCode)
	}
}

func TestPostMarkRead_400OnBadJSON(t *testing.T) {
	srv := newTestServer(t, &mockClient{}, testCfg())
	defer srv.Close()

	body := bytes.NewBufferString(`not json`)
	resp, err := http.Post(srv.URL+"/api/mark-read", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /api/mark-read bad body: got %d, want 400", resp.StatusCode)
	}
}

func TestPostMarkUnread_204(t *testing.T) {
	mc := &mockClient{unread: []miniflux.FeedEntry{makeEntry(1)}}
	srv := newTestServer(t, mc, testCfg())
	defer srv.Close()

	body := bytes.NewBufferString(`{"ids":[1]}`)
	resp, err := http.Post(srv.URL+"/api/mark-unread", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("POST /api/mark-unread: got %d, want 204", resp.StatusCode)
	}
}

func TestPostMarkUnread_400OnBadJSON(t *testing.T) {
	srv := newTestServer(t, &mockClient{}, testCfg())
	defer srv.Close()

	body := bytes.NewBufferString(`not json`)
	resp, err := http.Post(srv.URL+"/api/mark-unread", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /api/mark-unread bad body: got %d, want 400", resp.StatusCode)
	}
}

func TestPostToggleStar_204(t *testing.T) {
	mc := &mockClient{unread: []miniflux.FeedEntry{makeEntry(1)}}
	srv := newTestServer(t, mc, testCfg())
	defer srv.Close()

	body := bytes.NewBufferString(`{"id":1}`)
	resp, err := http.Post(srv.URL+"/api/toggle-star", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("POST /api/toggle-star: got %d, want 204", resp.StatusCode)
	}
}

func TestPostToggleStar_400OnBadJSON(t *testing.T) {
	srv := newTestServer(t, &mockClient{}, testCfg())
	defer srv.Close()

	body := bytes.NewBufferString(`not json`)
	resp, err := http.Post(srv.URL+"/api/toggle-star", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /api/toggle-star bad body: got %d, want 400", resp.StatusCode)
	}
}

func TestGetConfig_200(t *testing.T) {
	cfg := testCfg()
	cfg.ServerUrl = "http://miniflux.example.com"
	srv := newTestServer(t, &mockClient{}, cfg)
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/config")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/config: got %d, want 200", resp.StatusCode)
	}
	var got config.Config
	if err := json.NewDecoder(resp.Body).Decode(&got); err != nil {
		t.Fatal(err)
	}
	if got.ServerUrl != cfg.ServerUrl {
		t.Errorf("server_url: got %q, want %q", got.ServerUrl, cfg.ServerUrl)
	}
}

func TestPostConfig_204AndPersists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("DATA_DIR", tmpDir)

	srv := newTestServer(t, &mockClient{}, testCfg())
	defer srv.Close()

	incoming := config.Config{
		ApiKey:          "new-key",
		ServerUrl:       "http://new.example.com",
		CacheExpiryDays: 7,
	}
	buf, err := json.Marshal(incoming)
	if err != nil {
		t.Fatal(err)
	}
	resp, err := http.Post(srv.URL+"/api/config", "application/json", bytes.NewReader(buf))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusNoContent {
		t.Fatalf("POST /api/config: got %d, want 204", resp.StatusCode)
	}

	// Verify the config file was written.
	if _, err := os.Stat(tmpDir + "/config.toml"); err != nil {
		t.Errorf("config.toml not written: %v", err)
	}
}

func TestPostConfig_400OnBadJSON(t *testing.T) {
	srv := newTestServer(t, &mockClient{}, testCfg())
	defer srv.Close()

	body := bytes.NewBufferString(`not json`)
	resp, err := http.Post(srv.URL+"/api/config", "application/json", body)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("POST /api/config bad body: got %d, want 400", resp.StatusCode)
	}
}

func TestGetSearch_200WithResults(t *testing.T) {
	mc := &mockClient{search: []miniflux.FeedEntry{makeEntry(1), makeEntry(2)}}
	srv := newTestServer(t, mc, testCfg())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/search?q=foo")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/search: got %d, want 200", resp.StatusCode)
	}
	var result app.FetchResult
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatal(err)
	}
	if len(result.Entries) != 2 {
		t.Errorf("got %d entries, want 2", len(result.Entries))
	}
	if mc.searchQuery != "foo" {
		t.Errorf("search query: got %q, want %q", mc.searchQuery, "foo")
	}
}

func TestGetSearch_EmptyQuery_200(t *testing.T) {
	mc := &mockClient{search: []miniflux.FeedEntry{}}
	srv := newTestServer(t, mc, testCfg())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/search")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /api/search (no q): got %d, want 200", resp.StatusCode)
	}
}

func TestInvalidMethod_405(t *testing.T) {
	srv := newTestServer(t, &mockClient{}, testCfg())
	defer srv.Close()

	// Go 1.22+ method-specific routes return 405 for wrong methods.
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/api/cached", nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("DELETE /api/cached: got %d, want 405", resp.StatusCode)
	}
}

func TestGetFetchContent_400OnMissingID(t *testing.T) {
	srv := newTestServer(t, &mockClient{}, testCfg())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/api/fetch-content?url=http://example.com")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("GET /api/fetch-content missing id: got %d, want 400", resp.StatusCode)
	}
}

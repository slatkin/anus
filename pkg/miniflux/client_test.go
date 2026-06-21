package miniflux

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// newTestClient creates a Client pointing at the given test server URL.
func newTestClient(serverURL string) *Client {
	return NewClient(serverURL, "test-api-key", false)
}

// marshalJSON marshals v and panics on error (test helper).
func marshalJSON(v interface{}) []byte {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

// --- NewClient ---

func TestNewClient(t *testing.T) {
	c := NewClient("http://example.com", "mykey", false)
	if c.baseURL != "http://example.com" {
		t.Errorf("baseURL = %q, want %q", c.baseURL, "http://example.com")
	}
	if c.apiKey != "mykey" {
		t.Errorf("apiKey = %q, want %q", c.apiKey, "mykey")
	}
	if c.httpClient == nil {
		t.Error("httpClient is nil")
	}
}

func TestNewClientAllowInvalidCerts(t *testing.T) {
	// Just verify construction succeeds with TLS flag set.
	c := NewClient("https://example.com", "key", true)
	if c == nil {
		t.Fatal("expected non-nil client")
	}
}

// --- doRequest: auth header and network error ---

func TestDoRequestSetsAuthHeader(t *testing.T) {
	var gotKey string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotKey = r.Header.Get("X-Auth-Token")
		w.WriteHeader(http.StatusOK)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, _ = c.doRequest("GET", "/v1/entries", nil)
	if gotKey != "test-api-key" {
		t.Errorf("X-Auth-Token = %q, want %q", gotKey, "test-api-key")
	}
}

func TestDoRequestNetworkError(t *testing.T) {
	// Point at a closed server to trigger a network error.
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.doRequest("GET", "/v1/entries", nil)
	if err == nil {
		t.Error("expected error for closed server, got nil")
	}
}

func TestDoRequest4xxReturnsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.doRequest("GET", "/v1/entries", nil)
	if err == nil {
		t.Error("expected error for 403, got nil")
	}
	if !strings.Contains(err.Error(), "403") {
		t.Errorf("error should mention status 403: %v", err)
	}
}

func TestDoRequest5xxReturnsError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.doRequest("GET", "/v1/entries", nil)
	if err == nil {
		t.Error("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("error should mention status 500: %v", err)
	}
}

// --- GetUnreadEntries ---

func TestGetUnreadEntriesSuccess(t *testing.T) {
	entries := []FeedEntry{
		{ID: 1, Title: "First", Status: ReadStatusUnread, Content: "hello"},
		{ID: 2, Title: "Second", Status: ReadStatusUnread, Content: "world"},
	}
	resp := FeedEntriesResponse{Total: 2, Entries: entries}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/v1/entries") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(marshalJSON(resp))
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	got, total, err := c.GetUnreadEntries(10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
	if len(got) != 2 {
		t.Errorf("len(entries) = %d, want 2", len(got))
	}
}

func TestGetUnreadEntriesServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, _, err := c.GetUnreadEntries(10, 0)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- GetReadEntries ---

func TestGetReadEntriesSuccess(t *testing.T) {
	entries := []FeedEntry{{ID: 3, Title: "Old", Status: ReadStatusRead}}
	resp := FeedEntriesResponse{Total: 1, Entries: entries}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(marshalJSON(resp))
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	got, total, err := c.GetReadEntries(time.Now().Add(-24*time.Hour), 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 {
		t.Errorf("total = %d, want 1", total)
	}
	if len(got) != 1 || got[0].ID != 3 {
		t.Errorf("unexpected entries: %+v", got)
	}
}

func TestGetReadEntriesServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusBadGateway)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, _, err := c.GetReadEntries(time.Now(), 10, 0)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- GetStarredEntries ---

func TestGetStarredEntriesSuccess(t *testing.T) {
	entries := []FeedEntry{{ID: 5, Title: "Starred", Starred: true}}
	resp := FeedEntriesResponse{Total: 1, Entries: entries}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("starred") != "true" {
			t.Errorf("expected starred=true query param, got: %s", r.URL.RawQuery)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(marshalJSON(resp))
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	got, total, err := c.GetStarredEntries(10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(got) != 1 || got[0].ID != 5 {
		t.Errorf("unexpected result: total=%d entries=%+v", total, got)
	}
}

func TestGetStarredEntriesServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusUnauthorized)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, _, err := c.GetStarredEntries(10, 0)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- SearchEntries ---

func TestSearchEntriesSuccess(t *testing.T) {
	entries := []FeedEntry{{ID: 10, Title: "Match"}}
	resp := FeedEntriesResponse{Total: 1, Entries: entries}
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("search") == "" {
			t.Error("expected search query param")
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(marshalJSON(resp))
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	got, total, err := c.SearchEntries("golang", 10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if total != 1 || len(got) != 1 {
		t.Errorf("unexpected result: total=%d entries=%+v", total, got)
	}
}

func TestSearchEntriesServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, _, err := c.SearchEntries("query", 10, 0)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- ChangeEntryReadStatus (MarkRead / MarkUnread) ---

func TestChangeEntryReadStatusMarkRead(t *testing.T) {
	var gotBody UpdateEntriesRequest
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		if err := json.NewDecoder(r.Body).Decode(&gotBody); err != nil {
			t.Errorf("failed to decode body: %v", err)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	err := c.ChangeEntryReadStatus([]int{1, 2, 3}, ReadStatusRead)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotBody.Status != string(ReadStatusRead) {
		t.Errorf("status = %q, want %q", gotBody.Status, ReadStatusRead)
	}
	if len(gotBody.EntryIDs) != 3 {
		t.Errorf("entry_ids = %v, want [1 2 3]", gotBody.EntryIDs)
	}
}

func TestChangeEntryReadStatusMarkUnread(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	err := c.ChangeEntryReadStatus([]int{7}, ReadStatusUnread)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestChangeEntryReadStatusServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	err := c.ChangeEntryReadStatus([]int{1}, ReadStatusRead)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- ToggleStarred ---

func TestToggleStarredSuccess(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		if r.Method != http.MethodPut {
			t.Errorf("expected PUT, got %s", r.Method)
		}
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	err := c.ToggleStarred(42)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "/v1/entries/42/bookmark"
	if gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
}

func TestToggleStarredServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	err := c.ToggleStarred(42)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- SaveEntry ---

func TestSaveEntrySuccess(t *testing.T) {
	var gotPath string
	var gotMethod string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	err := c.SaveEntry(99)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := "/v1/entries/99/save"
	if gotPath != want {
		t.Errorf("path = %q, want %q", gotPath, want)
	}
	if gotMethod != http.MethodPost {
		t.Errorf("method = %q, want POST", gotMethod)
	}
}

func TestSaveEntryServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	err := c.SaveEntry(99)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- MarkAllAsRead ---

func TestMarkAllAsReadSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	err := c.MarkAllAsRead([]int{1, 2, 3})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- RefreshAllFeeds ---

func TestRefreshAllFeedsSuccess(t *testing.T) {
	var gotPath string
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	err := c.RefreshAllFeeds()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if gotPath != "/v1/feeds/refresh" {
		t.Errorf("path = %q, want /v1/feeds/refresh", gotPath)
	}
}

func TestRefreshAllFeedsServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	err := c.RefreshAllFeeds()
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- FetchOriginalContent ---

func TestFetchOriginalContentSuccess(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := OriginalContentResponse{Content: "<p>full article</p>"}
		w.Header().Set("Content-Type", "application/json")
		w.Write(marshalJSON(resp))
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	got, err := c.FetchOriginalContent(7)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != "<p>full article</p>" {
		t.Errorf("content = %q, want %q", got, "<p>full article</p>")
	}
}

func TestFetchOriginalContentServerError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "fail", http.StatusNotFound)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.FetchOriginalContent(7)
	if err == nil {
		t.Error("expected error, got nil")
	}
}

// --- fixProxyURLs (tested indirectly via GetUnreadEntries) ---

func TestFixProxyURLsRewritesLocalhostToServerURL(t *testing.T) {
	// Simulate a Miniflux response whose Content includes an http://localhost proxy URL.
	entries := []FeedEntry{
		{
			ID:      1,
			Title:   "Proxy test",
			Content: `<img src="http://localhost/proxy/abc123">`,
		},
	}
	resp := FeedEntriesResponse{Total: 1, Entries: entries}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(marshalJSON(resp))
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	got, _, err := c.GetUnreadEntries(10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got))
	}

	wantContent := fmt.Sprintf(`<img src="%s/proxy/abc123">`, ts.URL)
	if got[0].Content != wantContent {
		t.Errorf("content = %q\nwant    = %q", got[0].Content, wantContent)
	}
}

func TestFixProxyURLsDoesNotModifyNonLocalhostURLs(t *testing.T) {
	original := `<img src="https://cdn.example.com/image.png">`
	entries := []FeedEntry{{ID: 2, Content: original}}
	resp := FeedEntriesResponse{Total: 1, Entries: entries}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(marshalJSON(resp))
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	got, _, err := c.GetUnreadEntries(10, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got[0].Content != original {
		t.Errorf("content modified unexpectedly: %q", got[0].Content)
	}
}

func TestFixProxyURLsMultipleEntries(t *testing.T) {
	entries := []FeedEntry{
		{ID: 1, Content: `<img src="http://localhost/proxy/a">`},
		{ID: 2, Content: `<img src="http://localhost/proxy/b">`},
		{ID: 3, Content: `no proxy here`},
	}
	resp := FeedEntriesResponse{Total: 3, Entries: entries}

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write(marshalJSON(resp))
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	got, _, err := c.GetUnreadEntries(30, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	for i, e := range got {
		if strings.Contains(e.Content, "http://localhost/proxy/") {
			t.Errorf("entry %d still contains localhost proxy URL: %q", i, e.Content)
		}
	}
	if got[2].Content != "no proxy here" {
		t.Errorf("entry 2 content changed unexpectedly: %q", got[2].Content)
	}
}

// --- TLS / AllowInvalidCerts ---

func TestAllowInvalidCertsTLSServer(t *testing.T) {
	// Start a TLS test server with a self-signed cert.
	ts := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := FeedEntriesResponse{Total: 0, Entries: []FeedEntry{}}
		w.Header().Set("Content-Type", "application/json")
		w.Write(marshalJSON(resp))
	}))
	defer ts.Close()

	// Without AllowInvalidCerts, request should fail cert verification.
	cStrict := NewClient(ts.URL, "key", false)
	_, _, err := cStrict.GetUnreadEntries(10, 0)
	if err == nil {
		t.Error("expected TLS cert error for strict client, got nil")
	}

	// With AllowInvalidCerts=true, request should succeed.
	cPermissive := NewClient(ts.URL, "key", true)
	_, total, err := cPermissive.GetUnreadEntries(10, 0)
	if err != nil {
		t.Errorf("expected success with AllowInvalidCerts=true, got: %v", err)
	}
	if total != 0 {
		t.Errorf("total = %d, want 0", total)
	}
}

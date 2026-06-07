package app

import (
	"bytes"
	"fmt"
	"sort"
	"sync"
	"time"

	readability "codeberg.org/readeck/go-readability/v2"
	"github.com/slatkin/anus/internal/cache"
	"github.com/slatkin/anus/pkg/miniflux"
)

// MinifluxClient is the subset of miniflux.Client used by App, exposed as an
// interface so tests can substitute a mock.
type MinifluxClient interface {
	GetUnreadEntries(limit, offset int) ([]miniflux.FeedEntry, int, error)
	GetReadEntries(since time.Time, limit, offset int) ([]miniflux.FeedEntry, int, error)
	ChangeEntryReadStatus(ids []int, status miniflux.ReadStatus) error
	ToggleStarred(id int) error
	SaveEntry(id int) error
	RefreshAllFeeds() error
}

type App struct {
	client          MinifluxClient
	cache           *cache.Cache
	cacheExpiryDays int
	articleCache    map[int]string
	mu              sync.Mutex
}

// New creates an App. Call Open before use and Close when done.
func New(client MinifluxClient, cacheExpiryDays int) *App {
	return &App{
		client:          client,
		cacheExpiryDays: cacheExpiryDays,
		articleCache:    make(map[int]string),
	}
}

// Open initialises the cache at the given directory.
func (a *App) Open(cacheDir string) error {
	c, err := cache.Open(cacheDir, a.cacheExpiryDays)
	if err != nil {
		return fmt.Errorf("cache unavailable: %w", err)
	}
	a.cache = c
	_ = a.cache.Purge()
	return nil
}

func (a *App) Close() {
	if a.cache != nil {
		a.cache.Close()
	}
}

// ── types ─────────────────────────────────────────────────────────────────

type FeedSummary struct {
	FeedID    int    `json:"feed_id"`
	FeedTitle string `json:"feed_title"`
	Unread    int    `json:"unread"`
}

type FetchResult struct {
	Entries []miniflux.FeedEntry `json:"entries"`
	Feeds   []FeedSummary        `json:"feeds"`
}

// ── methods ───────────────────────────────────────────────────────────────

// FetchCached returns whatever is in the local cache without hitting the network.
func (a *App) FetchCached() (*FetchResult, error) {
	if a.cache == nil {
		return &FetchResult{}, nil
	}
	cached, err := a.cache.All()
	if err != nil {
		return nil, err
	}
	sortByDate(cached)
	return &FetchResult{Entries: cached, Feeds: buildFeedList(cached)}, nil
}

func (a *App) FetchEntries() (*FetchResult, error) {
	const pageSize = 100
	var fresh []miniflux.FeedEntry
	var fetchErr error

	for offset := 0; ; {
		entries, total, err := a.client.GetUnreadEntries(pageSize, offset)
		if err != nil {
			fetchErr = err
			break
		}
		fresh = append(fresh, entries...)
		offset += len(entries)
		if offset >= total || len(entries) == 0 {
			break
		}
	}

	if fetchErr == nil {
		since := time.Now().AddDate(0, 0, -30)
		for offset := 0; ; {
			entries, total, err := a.client.GetReadEntries(since, pageSize, offset)
			if err != nil {
				break
			}
			fresh = append(fresh, entries...)
			offset += len(entries)
			if offset >= total || len(entries) == 0 {
				break
			}
		}
	}

	if fetchErr != nil {
		if a.cache == nil {
			return nil, fetchErr
		}
		cached, err := a.cache.All()
		if err != nil || len(cached) == 0 {
			return nil, fetchErr
		}
		sortByDate(cached)
		return &FetchResult{Entries: cached, Feeds: buildFeedList(cached)}, nil
	}

	merged := make([]miniflux.FeedEntry, len(fresh))
	copy(merged, fresh)
	if a.cache != nil {
		freshSet := make(map[int]bool, len(fresh))
		freshFeedMap := make(map[int]miniflux.Feed, len(fresh))
		for _, e := range fresh {
			freshSet[e.ID] = true
			freshFeedMap[e.FeedID] = e.Feed
		}
		cached, _ := a.cache.All()
		fetchedAtMap := make(map[int]time.Time, len(cached))
		for _, e := range cached {
			fetchedAtMap[e.ID] = e.FetchedAt
		}
		now := time.Now()
		for i := range merged {
			if t, ok := fetchedAtMap[merged[i].ID]; ok {
				merged[i].FetchedAt = t
			} else {
				merged[i].FetchedAt = now
			}
		}
		_ = a.cache.Put(merged[:len(fresh)])
		var toReCache []miniflux.FeedEntry
		for _, e := range cached {
			if !freshSet[e.ID] {
				// Propagate current feed metadata (including category) to stale cached
				// entries that may have been stored before the Category field existed.
				if e.Feed.Category.ID == 0 {
					if f, ok := freshFeedMap[e.FeedID]; ok {
						e.Feed = f
						toReCache = append(toReCache, e)
					}
				}
				merged = append(merged, e)
			}
		}
		if len(toReCache) > 0 {
			_ = a.cache.Put(toReCache)
		}
	}

	sortByDate(merged)
	return &FetchResult{Entries: merged, Feeds: buildFeedList(merged)}, nil
}

func (a *App) RefreshAndFetch() (*FetchResult, error) {
	_ = a.client.RefreshAllFeeds()
	return a.FetchEntries()
}

// ClearCache wipes the local entry cache and fetches fresh data from Miniflux.
func (a *App) ClearCache() (*FetchResult, error) {
	if a.cache != nil {
		if err := a.cache.Clear(); err != nil {
			return nil, fmt.Errorf("clear cache: %w", err)
		}
	}
	return a.FetchEntries()
}

func (a *App) FetchArticleContent(id int, url string) (string, error) {
	a.mu.Lock()
	if html, ok := a.articleCache[id]; ok {
		a.mu.Unlock()
		return html, nil
	}
	a.mu.Unlock()

	article, err := readability.FromURL(url, 30*time.Second)
	if err != nil {
		return "", err
	}
	var buf bytes.Buffer
	if err := article.RenderHTML(&buf); err != nil {
		return "", err
	}
	html := buf.String()

	a.mu.Lock()
	a.articleCache[id] = html
	a.mu.Unlock()
	return html, nil
}

func (a *App) MarkRead(ids []int) error {
	if a.cache != nil {
		for _, id := range ids {
			_ = a.cache.Update(id, func(e *miniflux.FeedEntry) { e.Status = miniflux.ReadStatusRead })
		}
	}
	return a.client.ChangeEntryReadStatus(ids, miniflux.ReadStatusRead)
}

func (a *App) MarkUnread(ids []int) error {
	if a.cache != nil {
		for _, id := range ids {
			_ = a.cache.Update(id, func(e *miniflux.FeedEntry) { e.Status = miniflux.ReadStatusUnread })
		}
	}
	return a.client.ChangeEntryReadStatus(ids, miniflux.ReadStatusUnread)
}

func (a *App) ToggleStar(id int) error {
	if a.cache != nil {
		_ = a.cache.Update(id, func(e *miniflux.FeedEntry) { e.Starred = !e.Starred })
	}
	return a.client.ToggleStarred(id)
}

func (a *App) SaveEntry(id int) error {
	return a.client.SaveEntry(id)
}

func ApplyConfig(a *App, cacheExpiryDays int) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.cacheExpiryDays = cacheExpiryDays
}

// ── helpers ───────────────────────────────────────────────────────────────

func sortByDate(entries []miniflux.FeedEntry) {
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].PublishedAt.After(entries[j].PublishedAt)
	})
}

func buildFeedList(entries []miniflux.FeedEntry) []FeedSummary {
	type feedData struct {
		title  string
		unread int
	}
	byID := make(map[int]*feedData)
	var order []int

	for _, e := range entries {
		if _, ok := byID[e.FeedID]; !ok {
			byID[e.FeedID] = &feedData{title: e.Feed.Title}
			order = append(order, e.FeedID)
		}
		if e.Status == miniflux.ReadStatusUnread {
			byID[e.FeedID].unread++
		}
	}

	result := make([]FeedSummary, 0, len(byID))
	for _, id := range order {
		d := byID[id]
		result = append(result, FeedSummary{FeedID: id, FeedTitle: d.title, Unread: d.unread})
	}

	sort.Slice(result, func(i, j int) bool {
		if result[i].Unread != result[j].Unread {
			return result[i].Unread > result[j].Unread
		}
		return result[i].FeedTitle < result[j].FeedTitle
	})

	return result
}

package app

import (
	"bytes"
	"fmt"
	"log"
	"sort"
	"sync"
	"time"

	readability "codeberg.org/readeck/go-readability/v2"
	"golang.org/x/sync/singleflight"

	"github.com/slatkin/anus/internal/cache"
	"github.com/slatkin/anus/pkg/miniflux"
)

// MinifluxClient is the subset of miniflux.Client used by App, exposed as an
// interface so tests can substitute a mock.
type MinifluxClient interface {
	GetUnreadEntries(limit, offset int) ([]miniflux.FeedEntry, int, error)
	GetReadEntries(since time.Time, limit, offset int) ([]miniflux.FeedEntry, int, error)
	SearchEntries(query string, limit, offset int) ([]miniflux.FeedEntry, int, error)
	ChangeEntryReadStatus(ids []int, status miniflux.ReadStatus) error
	ToggleStarred(id int) error
	SaveEntry(id int) error
	RefreshAllFeeds() error
}

type App struct {
	client          MinifluxClient
	cache           *cache.Cache
	cacheExpiryDays int
	articleCache    sync.Map
	articleFlight   singleflight.Group
	mu              sync.Mutex
}

// New creates an App. Call Open before use and Close when done.
func New(client MinifluxClient, cacheExpiryDays int) *App {
	return &App{
		client:          client,
		cacheExpiryDays: cacheExpiryDays,
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
	fresh, fetchErr := paginate(func(limit, offset int) ([]miniflux.FeedEntry, int, error) {
		return a.client.GetUnreadEntries(limit, offset)
	})
	if fetchErr == nil {
		since := time.Now().AddDate(0, 0, -30)
		read, _ := paginate(func(limit, offset int) ([]miniflux.FeedEntry, int, error) {
			return a.client.GetReadEntries(since, limit, offset)
		})
		fresh = append(fresh, read...)
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

	merged := fresh
	if a.cache != nil {
		merged = a.mergeWithCache(fresh)
	}
	sortByDate(merged)
	return &FetchResult{Entries: merged, Feeds: buildFeedList(merged)}, nil
}

func (a *App) SearchEntries(query string) (*FetchResult, error) {
	all, err := paginate(func(limit, offset int) ([]miniflux.FeedEntry, int, error) {
		return a.client.SearchEntries(query, limit, offset)
	})
	if err != nil {
		return nil, err
	}
	sortByDate(all)
	return &FetchResult{Entries: all, Feeds: buildFeedList(all)}, nil
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
	if v, ok := a.articleCache.Load(id); ok {
		return v.(string), nil
	}
	v, err, _ := a.articleFlight.Do(fmt.Sprintf("%d", id), func() (interface{}, error) {
		article, err := readability.FromURL(url, 30*time.Second)
		if err != nil {
			return "", err
		}
		var buf bytes.Buffer
		if err := article.RenderHTML(&buf); err != nil {
			return "", err
		}
		html := buf.String()
		a.articleCache.Store(id, html)
		return html, nil
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func (a *App) MarkRead(ids []int) error {
	if a.cache != nil {
		for _, id := range ids {
			if err := a.cache.Update(id, func(e *miniflux.FeedEntry) { e.Status = miniflux.ReadStatusRead }); err != nil {
				log.Printf("cache Update: %v", err)
			}
		}
	}
	return a.client.ChangeEntryReadStatus(ids, miniflux.ReadStatusRead)
}

func (a *App) MarkUnread(ids []int) error {
	if a.cache != nil {
		for _, id := range ids {
			if err := a.cache.Update(id, func(e *miniflux.FeedEntry) { e.Status = miniflux.ReadStatusUnread }); err != nil {
				log.Printf("cache Update: %v", err)
			}
		}
	}
	return a.client.ChangeEntryReadStatus(ids, miniflux.ReadStatusUnread)
}

func (a *App) ToggleStar(id int) error {
	if a.cache != nil {
		if err := a.cache.Update(id, func(e *miniflux.FeedEntry) { e.Starred = !e.Starred }); err != nil {
			log.Printf("cache Update: %v", err)
		}
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

// paginate calls fetch repeatedly, advancing offset, until all pages are collected.
func paginate(fetch func(limit, offset int) ([]miniflux.FeedEntry, int, error)) ([]miniflux.FeedEntry, error) {
	const pageSize = 100
	var all []miniflux.FeedEntry
	for offset := 0; ; {
		entries, total, err := fetch(pageSize, offset)
		if err != nil {
			return nil, err
		}
		all = append(all, entries...)
		offset += len(entries)
		if offset >= total || len(entries) == 0 {
			break
		}
	}
	return all, nil
}

// mergeWithCache combines fresh API entries with cached entries, preserving
// FetchedAt timestamps and propagating up-to-date feed metadata to stale entries.
func (a *App) mergeWithCache(fresh []miniflux.FeedEntry) []miniflux.FeedEntry {
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

	merged := make([]miniflux.FeedEntry, len(fresh))
	copy(merged, fresh)
	now := time.Now()
	for i := range merged {
		if t, ok := fetchedAtMap[merged[i].ID]; ok {
			merged[i].FetchedAt = t
		} else {
			merged[i].FetchedAt = now
		}
	}
	if err := a.cache.Put(merged); err != nil {
		log.Printf("cache Put: %v", err)
	}

	var toReCache []miniflux.FeedEntry
	for _, e := range cached {
		if freshSet[e.ID] {
			continue
		}
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
	if len(toReCache) > 0 {
		if err := a.cache.Put(toReCache); err != nil {
			log.Printf("cache Put: %v", err)
		}
	}
	return merged
}

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

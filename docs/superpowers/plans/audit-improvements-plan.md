# Audit Improvements Plan

## Global Constraints
- Working directory: /home/slatkin/Dev/anus
- Go module: follows existing patterns in pkg/
- Frontend: Svelte 5, Vite, Vitest for tests
- NEVER add Co-Authored-By to commits
- DO ONLY what was asked, no extra features
- Make code files small and modular
- No monoliths

## Task 1: Add unit tests for pkg/miniflux/client.go

Add HTTP mock tests for `pkg/miniflux/client.go` using Go's `net/http/httptest` package.

File to create: `pkg/miniflux/client_test.go`

Cover:
- `NewClient()` construction
- `GetEntries()` success and error paths (4xx, 5xx, network error)
- `GetEntry()` success and error paths
- `Search()` success and error paths
- `MarkRead()`, `MarkUnread()`, `ToggleStarred()`, `SaveEntry()` success + error paths
- `fixProxyURLs()` rewriting logic (localhost→server URL)
- TLS/`AllowInvalidCerts` config

Use `httptest.NewServer` with handler stubs returning JSON. Do NOT use external mocking libraries — only stdlib. Run `go test ./pkg/miniflux/...` to verify.

Commit message: "Add unit tests for miniflux HTTP client"

## Task 2: Add unit tests for frontend/src/utils/content.js

Add Vitest unit tests for `frontend/src/utils/content.js`.

File to create: `frontend/src/utils/content.test.js`

The test environment must use `jsdom` (set `testEnvironment: 'jsdom'` in vitest config or use a `@vitest-environment jsdom` comment). Check `frontend/vite.config.js` for how existing tests configure their environment.

Cover each transformation in `content.js`:
1. Removal of "View Image in Fullscreen" links
2. Stripping width/height from `<img>` tags
3. YouTube iframe → thumbnail conversion
4. figure/figcaption cleanup (duplicate sibling removal)
5. Caption deduplication
6. Bare image + text → `<figure>` wrapping
7. `<br>`-delimited → `<p>` conversion

Run `cd frontend && npm test` to verify all pass.

Commit message: "Add unit tests for content.js HTML sanitization"

## Task 3: Add handler tests for cmd/anus-web/main.go

Add HTTP handler tests for `cmd/anus-web/main.go` using `net/http/httptest`.

File to create: `cmd/anus-web/main_test.go`

The web server uses `pkg/app` (App struct). You'll need to construct a test App or use a mock. Look at how `pkg/app/app_test.go` sets up mocks (it uses a `mockMinifluxClient`).

Cover:
- `GET /api/cached` → 200 with JSON entries
- `GET /api/entries` → 200 (calls FetchAndCache)
- `POST /api/mark-read` → 200
- `POST /api/mark-unread` → 200
- `POST /api/toggle-star` → 200
- `GET /api/config` → 200 with sanitized config (no api_key)
- `POST /api/config` → 200, persists config
- `GET /api/search?q=foo` → 200 with results
- Invalid method → 405 or appropriate error
- Missing params → 400

Run `go test ./cmd/anus-web/...` to verify all pass.

Commit message: "Add HTTP handler tests for anus-web"

## Task 4: Extract ReaderPane and FeedList from App.svelte

Split `frontend/src/App.svelte` into smaller components.

Extract two components:
1. `frontend/src/components/ReaderPane.svelte` — the multi-column reader layout: the content div, column layout/pagination display, and the reader controls toolbar row at the bottom. Props: `entry`, `pages`, `currentPage`, events: `next-page`, `prev-page`, `mark-read`, `toggle-star`, `save`, `open-browser`, `close`.
2. `frontend/src/components/FeedList.svelte` — the left nav panel: the list of feed items and headers. Props: `items` (the grouped/flat item array), `cursor`, `selectedFeedId`, events: `select`, `collapse-toggle`.

App.svelte retains: data fetching, state stores, keyboard handling, mode switching, and composition of the layout.

Existing extracted components (NavItem, NavFeedHeader, etc.) must continue to work — FeedList should use them internally.

Do NOT change any visual behavior or styling. The app must look and function identically after extraction.

Run `cd frontend && npm test` to verify no regressions. Manually verify the dev server starts: `make dev-web` (check it compiles without error, you don't need to open a browser).

Commit message: "Extract ReaderPane and FeedList components from App.svelte"

## Task 5: Add AbortController timeout to api.web.js fetch calls

In `frontend/src/api.web.js`, add a 15-second `AbortController` timeout to all `fetch` calls.

The existing `post()` helper should gain a timeout. For GET-style calls (if any use raw fetch), apply the same pattern. Use `AbortSignal.timeout(15000)` if available in the target environment, otherwise use `AbortController` + `setTimeout`. Check what Vite/browser targets are configured.

Do NOT change the function signatures or add new exports.

Run `cd frontend && npm test` to verify no regressions.

Commit message: "Add 15s fetch timeout to api.web.js"

## Task 6: Consolidate config-save logic

Both `main.go` (Wails desktop) and `cmd/anus-web/main.go` (web server) contain config-save logic. Extract it to a shared helper.

Create `pkg/config/save.go` with a `Save(cfg Config, path string) error` function that:
- Encodes the config to TOML
- Writes it atomically (write to temp file, rename) or directly to the path
- Returns any error

Update `main.go` and `cmd/anus-web/main.go` to call `config.Save(...)` instead of their inline logic.

Run `go test ./...` to verify no regressions.

Commit message: "Extract config save logic to pkg/config/save.go"

## Task 7: Add error logging for suppressed cache write failures

In `pkg/app/app.go`, the `_ = cache.Put(...)`, `_ = cache.Update(...)` etc. calls silently swallow errors.

Add a simple `log.Printf` (using stdlib `log` package, already imported or add the import) for each suppressed cache error. Format: `"cache %s: %v"` where the first arg is the operation name (e.g., `"Put"`, `"Update"`).

Do NOT change the non-blocking behavior — cache errors still do not propagate to callers.

Run `go test ./pkg/app/...` to verify no regressions.

Commit message: "Log suppressed cache errors in app.go"

## Task 8: Add localStorage version key to preferences.js

In `frontend/src/stores/preferences.js`, add a version migration guard.

Add a `PREFS_VERSION = 1` constant. On app startup (module load), check `localStorage.getItem('prefs_version')`. If it's missing or less than `PREFS_VERSION`, clear all known preference keys (list them explicitly — do not use `localStorage.clear()` which would blow away unrelated data). Then write `localStorage.setItem('prefs_version', PREFS_VERSION)`.

The list of keys to clear on version mismatch: all keys used by the existing `persisted()` stores in the file.

Run `cd frontend && npm test` to verify no regressions.

Commit message: "Add localStorage version guard to preferences.js"

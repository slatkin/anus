# Core — anus project

Miniflux RSS reader frontend with two deployment modes from one codebase.

## Deployment modes
- **Desktop (`anus-ui`):** Wails v2 + WebKit2GTK webview. Entry: `main.go`.
- **Web (`anus-web`):** Go HTTP server + static frontend. Entry: `cmd/anus-web/main.go`.

## Source map
```
main.go                  # Wails desktop entry
cmd/anus-web/            # Web server entry + dev/prod embed
pkg/app/app.go           # Core business logic (fetch, cache, search, mark read/starred)
pkg/miniflux/            # HTTP client wrapping Miniflux API
pkg/config/              # Config loading (defaults → env vars → TOML)
internal/cache/cache.go  # BoltDB persistence (cache-first fallback)
frontend/                # Svelte 5 SPA (see mem:frontend/core)
```

## Config file
`~/.config/anus/config.toml` — keys: `api_key`, `server_url`. Also readable from env vars `MINIFLUX_API_KEY`, `MINIFLUX_URL`.

## Module path
`github.com/slatkin/anus`

See `mem:frontend/core` for frontend details, `mem:tech_stack` for versions, `mem:suggested_commands` for build/dev commands.

# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Rules

**NEVER ADD CO-AUTHORED-BY.** Do not add `Co-Authored-By` trailers to commit messages.

**NEVER GUESS.** Read the source before assuming anything. Use Grep, Glob, or Read first.

**DO ONLY WHAT WAS ASKED.** No extra borders, styles, classes, or behaviours.

**NO MONOLITHS** Make code files small and modular, and logically separated by function. No giant files of code that fills up Claude's context when trying to read and locate code.

**DELEGATE TO SUBAGENTS** Try to deletgate simple code reads to subagents to avoids growing context in the main thread.

**DEBUG AND TROUBLESHOOT, DON'T SPIN YOUR WHEELS SPECULATING** Being direct and adding debugging and conducting tests to get more information about an issue is preferred over staring at the code for extended periods of time trying to speculate what might be happening.

**DON'T BE A DICK** When a bug fix does not resolve the issue, do NOT suspect user error. Assume the fix is wrong or incomplete and investigate the code further.

## Commands

```bash
# Development
make dev-ui    # Wails hot-reload desktop dev server (VITE_API=wails)
make dev-web   # Go API on :8080 + Vite dev server on :5173 (web mode)

# Testing
make test                                  # go test ./...
go test -run TestFetchEntries ./pkg/app    # single Go test
cd frontend && npm test                    # frontend tests (Vitest)
```

## Architecture

The app is a frontend for [Miniflux](https://miniflux.app/) RSS that runs in **two deployment modes** from the same codebase:

- **Desktop (anus-ui):** Wails v2 app — Go backend bound directly to the WebKit2GTK webview. Entry point: [main.go](main.go).
- **Web (anus-web):** Go HTTP server serving the frontend as static files + `/api/*` JSON endpoints. Entry point: [cmd/anus-web/main.go](cmd/anus-web/main.go).

### Frontend–Backend Connection

The Vite build aliases `./api.js` at build time based on the `VITE_API` env var:
- `VITE_API=wails` → [frontend/src/api.wails.js](frontend/src/api.wails.js) — calls Wails Go bindings (FFI)
- `VITE_API=web` → [frontend/src/api.web.js](frontend/src/api.web.js) — makes HTTP fetch calls

In web dev mode, Vite proxies `/api/*` to the Go server on :8080.

### Frontend Structure

[frontend/src/App.svelte](frontend/src/App.svelte) is the root component — owns core data (`entries`, `selectedEntry`, `feeds`), navigation logic, and keyboard handling. Everything else is broken out:

- [frontend/src/components/](frontend/src/components/) — `NavItem`, `NavFeedHeader`, `SearchBar`, `ReaderControls`, `PageNav`, `Toast`, `SettingsModal`
- [frontend/src/stores/preferences.js](frontend/src/stores/preferences.js) — all `localStorage`-persisted UI state as Svelte writable stores
- [frontend/src/utils/date.js](frontend/src/utils/date.js) / [content.js](frontend/src/utils/content.js) — date formatting and HTML processing helpers
- [frontend/src/grouping.js](frontend/src/grouping.js) / [paging.js](frontend/src/paging.js) — grouping and pagination logic with Vitest tests

### Key Go Packages

| Path | Purpose |
|------|---------|
| [pkg/app/](pkg/app/) | Core business logic: fetch, cache fallback, search, mark read/starred |
| [pkg/miniflux/](pkg/miniflux/) | HTTP client wrapping the Miniflux API |
| [pkg/config/](pkg/config/) | Config loading — defaults → env vars → TOML file (`~/.config/anus/config.toml`) |
| [internal/cache/](internal/cache/) | BoltDB persistence for entries (cache-first fallback when API is unreachable) |
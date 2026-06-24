# Frontend core

Svelte 5 SPA in `frontend/src/`. No SvelteKit — plain Vite build.

## Key files
```
App.svelte                    # Root: owns entries, selectedEntry, feeds, keyboard handling
components/NavItem.svelte
components/NavFeedHeader.svelte
components/SearchBar.svelte
components/ReaderControls.svelte
components/PageNav.svelte
components/Toast.svelte
components/SettingsModal.svelte
stores/preferences.js         # All localStorage-persisted UI state (Svelte writable stores)
utils/date.js                 # Date formatting helpers
utils/content.js              # HTML processing helpers
grouping.js / paging.js       # Grouping and pagination logic (have Vitest tests)
main.js                       # Entry point
```

## API layer (build-time aliased)
`./api.js` is resolved at build time via `VITE_API` env var:
- `VITE_API=wails` → `api.wails.js` (calls Wails Go FFI bindings)
- `VITE_API=web`   → `api.web.js` (HTTP fetch calls)

In `dev-web` mode, Vite proxies `/api/*` to Go server on :8080.

## Wails JS bindings
Generated bindings live in `frontend/wailsjs/` — do not edit manually.

## a11y
Fix all `vite-plugin-svelte` a11y warnings immediately; never let them accumulate (see user memory).

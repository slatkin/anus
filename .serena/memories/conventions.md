# Conventions

## Go
- Packages are small and focused — no giant files. `pkg/app/app.go` is the main business logic file.
- Config struct loaded once at startup; passed by value/pointer into app logic.
- Cache-first pattern: try Miniflux API, fall back to BoltDB cache on failure.

## Frontend (Svelte)
- Svelte 5 runes/reactivity (not legacy Svelte 3/4 `$:` syntax where avoidable).
- All persisted UI state lives in `stores/preferences.js` as writable stores backed by `localStorage`.
- No TypeScript — plain `.js` and `.svelte` files.
- Component files in `frontend/src/components/`, utilities in `frontend/src/utils/`.
- Logic with tests (`grouping.js`, `paging.js`) kept separate from components.
- `./api.js` import is always the virtual alias — never import `api.wails.js` or `api.web.js` directly.

## Commits
- No `Co-Authored-By` trailers. No attribution lines. (See user global memory.)
- Commit messages: concise, imperative, describe the why not the what.

## General
- Keep files small and modular; avoid monoliths.
- No speculative features or abstractions beyond the task.
- No comments unless the WHY is non-obvious.

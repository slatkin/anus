# Tech stack

## Backend (Go)
- Go 1.25 (`go.mod`)
- Wails v2.12.0 — desktop shell
- BoltDB (`go.etcd.io/bbolt` v1.4.3) — local cache
- Echo v4 (`labstack/echo`) — HTTP router for web mode
- `codeberg.org/readeck/go-readability/v2` — article extraction
- Build tag `webkit2_41` required for desktop builds (GTK WebKit version)

## Frontend
- Svelte 5.56+
- Vite 8
- Vitest 4 (unit tests)
- `lucide-svelte` — icons
- No TypeScript (plain JS + Svelte)

## Toolchain
- `wails` CLI — desktop dev/build
- `npm` (not yarn/pnpm) — frontend package manager
- `make` — primary task runner (see `mem:suggested_commands`)
- Docker available (`Dockerfile`, `docker-compose.yml`)

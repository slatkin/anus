# anus

**A newsreader, unfussy & simple.**

anus is a self-hosted [Miniflux](https://miniflux.app/) reader that runs as a web service in a Docker container. It was built with Go and Svelte. As I would never have the time to spend making something so niche on my own in my free time, this is a vibe-coding project and could possibly result in the end of human civilisation.

---

## anus-web (Docker)

### Quick start

```bash
docker run -d \
  -e MINIFLUX_URL=https://your-miniflux-instance \
  -e MINIFLUX_API_KEY=your-api-key \
  -v anus-data:/data \
  -p 8080:8080 \
  ghcr.io/slatkin/anus-web:latest
```

Or with `docker compose` (copy `docker-compose.yml.sample` to `docker-compose.yml` and fill in your credentials):

```yaml
services:
  anus-web:
    image: ghcr.io/slatkin/anus-web:latest
    restart: unless-stopped
    ports:
      - "8080:8080"
    environment:
      MINIFLUX_URL: https://your-miniflux-instance
      MINIFLUX_API_KEY: your-api-key
      DATA_DIR: /data
      CACHE_DIR: /data
    volumes:
      - anus-data:/data

volumes:
  anus-data:
```

```bash
docker compose up -d
```

### Getting your Miniflux API key

In your Miniflux instance: **Settings → API Keys → Create a new API key**. Copy the key and use it as `MINIFLUX_API_KEY`.

### Environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MINIFLUX_URL` | yes | — | Base URL of your Miniflux instance |
| `MINIFLUX_API_KEY` | yes | — | Miniflux API key (Settings → API Keys) |
| `DATA_DIR` | no | `/data` | Persistent data directory (config + cache); mount a volume here |
| `CACHE_DIR` | no | `DATA_DIR` | Override cache location (defaults to same as `DATA_DIR`) |
| `ALLOW_INVALID_CERTS` | no | `false` | Skip TLS verification |
| `CACHE_EXPIRY_DAYS` | no | `30` | Article cache retention in days |
| `PORT` | no | `8080` | HTTP listen port |

---

## Keyboard shortcuts

### Navigation

| Key | Action |
|-----|--------|
| `↑` / `↓` | Previous / next article |
| `Home` / `End` | Jump to first / last article |
| `Enter` | Open selected article |
| `→` | Next page in reader |
| `←` | Previous page in reader |
| `b` / `Esc` / `Backspace` | Back (reader → list → feeds) |
| `f` | Switch to feed list |

### Article actions

| Key | Action |
|-----|--------|
| `Space` | Mark current article read and advance to next unread |
| `u` / `m` | Toggle read / unread |
| `s` | Toggle starred |
| `A` | Mark all articles read |
| `e` | Save article to Miniflux read-later |
| `o` | Open article URL in browser |
| `r` | Refresh articles from Miniflux |
| `?` | Show shortcut hint in status bar |

### Search

| Key | Action |
|-----|--------|
| `/` | Open search |
| `Enter` | Run search |
| `Esc` | Close search and return to normal list |

Click the search icon in the bottom toolbar to open search with the mouse.

In search mode the article list shows only results from Miniflux's full-text search. Matched terms are highlighted in the reader. The search input turns green on results, red on no results. Press `Esc` or `b` to exit; the list scrolls back to the previously selected article, expanding its group and enabling the "all" filter if needed.

### Grouping

Use the **Group** button in the toolbar to switch between three list layouts:

| Mode | Description |
|------|-------------|
| Ungrouped | Flat list of all articles sorted by date |
| By feed | Articles grouped under their feed name |
| By category | Articles grouped under their Miniflux category |

In grouped modes, click a group header to collapse or expand it. The collapse-all / expand-all buttons above the list act on all groups at once. Groups are always expanded in search mode.

---

## Development

```bash
make dev        # Go API on :8080 + Vite dev server on :5173
make build      # frontend + production binary to build/bin/anus
make test       # go test ./...
make fmt        # gofmt
make vet        # go vet
make tidy       # go mod tidy
```

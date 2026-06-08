# anus

**A newsreader, unfussy & simple.**

anus is a [Miniflux](https://miniflux.app/) reader available as both a native Linux desktop app and a self-hosted web service. It was built with Go, Svelte, and [Wails v2](https://wails.io/). As I would never have the time to spend making something so niche on my own in my free time, this is a vibe-coding project and could possibly result in the end of human civilisation.

anus can be used two ways: as a local client or as a web service running in a docker container.

| | `anus-ui` | `anus-web` |
|---|---|---|
| **What** | Native desktop app (WebKit2GTK) | Containerised web service |
| **Install** | AUR (`anus`) or build from source | Docker / `docker compose` |
| **Config** | `~/.config/anus/config.toml` | Settings UI or environment variables, persisted to `/data/config.toml` |

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

## anus-ui (Desktop)

### Requirements

- Linux with WebKit2GTK 4.1 (`webkit2gtk-4.1`)
- [Miniflux](https://miniflux.app/) server with API access

### Installation

#### AUR (Arch / CachyOS)

```bash
# Binary release
paru -S anus

# Build from source
paru -S anus-git
```

#### Build from source

```bash
# Dependencies
sudo pacman -S wails webkit2gtk-4.1 gtk3 nodejs npm imagemagick

git clone https://github.com/slatkin/anus ~/Dev/anus
cd ~/Dev/anus
make build-ui          # builds to build/bin/anus
make install           # installs binary, .desktop file, and icons to ~/.local
```

### Configuration

Initialise a config file on first run:

```bash
anus --init
$EDITOR ~/.config/anus/config.toml
```

```toml
api_key            = "your-miniflux-api-key"
server_url         = "https://your-miniflux-instance.example.com"
allow_invalid_certs = false
cache_expiry_days  = 30
remember_read_position = true
```

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
make dev-ui     # Wails hot-reload dev server (desktop)
make dev-web    # Go API on :8080 + Vite dev server on :5173 (web)
make test       # go test ./...
make fmt        # gofmt
make vet        # go vet
make tidy       # go mod tidy
```

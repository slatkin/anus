# anus

**A Newsreader, Unfussy & Simple.**

A [Miniflux](https://miniflux.app/) reader available as both a native Linux desktop app and a self-hosted web service. Built with Go, Svelte, and [Wails v2](https://wails.io/).

## Variants

| | `anus-ui` | `anus-web` |
|---|---|---|
| **What** | Native desktop app (WebKit2GTK) | Containerised web service |
| **Install** | AUR (`anus-ui`) or build from source | Docker / `docker compose` |
| **Config** | `~/.config/anus/config.toml` | Environment variables |

## Features

- Reads unread articles and recent read articles (last 30 days) from your Miniflux server
- Multi-column paginated reader that adapts to window width (~560px per column)
- Rosé Pine Moon sidebar, sepia reader pane, LexendDeca font
- Local BoltDB article cache with configurable expiry; falls back to cache when offline
- Auto-polls Miniflux every 10 minutes for new articles
- YouTube embeds replaced with clickable thumbnails
- Collapsible sidebar

---

## anus-web (Docker)

### Quick start

```bash
docker run -d \
  -e MINIFLUX_URL=https://your-miniflux-instance \
  -e MINIFLUX_API_KEY=your-api-key \
  -v anus-cache:/data/cache \
  -e CACHE_DIR=/data/cache \
  -p 8080:8080 \
  ghcr.io/slatkin/anus-web:latest
```

Or with `docker compose`:

```bash
# edit docker-compose.yml to set MINIFLUX_URL and MINIFLUX_API_KEY
docker compose up -d
```

### Environment variables

| Variable | Required | Default | Description |
|----------|----------|---------|-------------|
| `MINIFLUX_URL` | yes | — | Base URL of your Miniflux instance |
| `MINIFLUX_API_KEY` | yes | — | Miniflux API key (Settings → API Keys) |
| `ALLOW_INVALID_CERTS` | no | `false` | Skip TLS verification |
| `CACHE_EXPIRY_DAYS` | no | `30` | Article cache retention in days |
| `CACHE_DIR` | no | `~/.cache/anus` | Path to cache directory (use a volume) |
| `REMEMBER_READ_POSITION` | no | `true` | Restore scroll position on revisit |
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
yay -S anus-ui

# Build from source
yay -S anus-ui-git
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
| `Enter` | Open selected article |
| `→` / `PageDown` | Next page; at last page, advance to next article |
| `←` / `PageUp` | Previous page; at first page, go to previous article |
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

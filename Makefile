BIN_UI  := anus
BIN_WEB := anus-web
TAGS    := webkit2_41
OUTDIR  := build/bin
PREFIX  ?= $(HOME)/.local

.PHONY: build-ui build-web dev-ui dev-web dev-docker test docker-build fmt vet tidy clean install uninstall

build-ui: frontend-wails
	wails build -tags $(TAGS) -o $(BIN_UI) .

build-web: frontend-web
	go build -tags production -o $(OUTDIR)/$(BIN_WEB) ./cmd/anus-web

dev-ui:
	VITE_API=wails wails dev -tags $(TAGS)

dev-docker:
	@CONFIG=~/.config/anus/config.toml; \
	APIKEY=$$(grep '^api_key' $$CONFIG | cut -d'"' -f2); \
	URL=$$(grep '^server_url' $$CONFIG | cut -d'"' -f2); \
	docker buildx build -t anus-web . && \
	docker rm -f anus-web-dev 2>/dev/null || true; \
	docker run --rm --name anus-web-dev -p 8888:8080 \
		-e MINIFLUX_API_KEY=$$APIKEY \
		-e MINIFLUX_URL=$$URL \
		anus-web

dev-web:
	@trap 'kill 0' EXIT; \
	CONFIG=~/.config/anus/config.toml; \
	export MINIFLUX_API_KEY=$$(grep '^api_key' $$CONFIG | cut -d'"' -f2); \
	export MINIFLUX_URL=$$(grep '^server_url' $$CONFIG | cut -d'"' -f2); \
	export CACHE_DIR=$$HOME/.cache/anus-web; \
	cd frontend && npm run dev & go run ./cmd/anus-web

frontend-wails:
	cd frontend && VITE_API=wails npm run build

frontend-web:
	cd frontend && VITE_API=web npm run build

test:
	go test ./...

docker-build: frontend-web
	docker buildx build -t anus-web .

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf build frontend/dist

ICON_SRC   := assets/appicon.png
ICON_SIZES := 16 32 48 64 128 256

install: build-ui
	install -Dm755 $(OUTDIR)/$(BIN_UI) $(PREFIX)/bin/$(BIN_UI)
	install -Dm644 anus.desktop $(PREFIX)/share/applications/anus.desktop
	for sz in $(ICON_SIZES); do \
		mkdir -p $(PREFIX)/share/icons/hicolor/$${sz}x$${sz}/apps && \
		magick $(ICON_SRC) -resize $${sz}x$${sz} $(PREFIX)/share/icons/hicolor/$${sz}x$${sz}/apps/$(BIN_UI).png; \
	done

uninstall:
	rm -f $(PREFIX)/bin/$(BIN_UI)
	rm -f $(PREFIX)/share/applications/anus.desktop
	for sz in $(ICON_SIZES); do \
		rm -f $(PREFIX)/share/icons/hicolor/$${sz}x$${sz}/apps/$(BIN_UI).png; \
	done

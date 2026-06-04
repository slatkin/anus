BIN_UI  := anus
BIN_WEB := anus-web
TAGS    := webkit2_41
OUTDIR  := build/bin
PREFIX  ?= $(HOME)/.local

.PHONY: build-ui build-web dev-ui dev-web test docker-build fmt vet tidy clean install uninstall

build-ui: frontend-wails
	wails build -tags $(TAGS) -o $(BIN_UI)

build-web: frontend-web
	go build -tags production -o $(OUTDIR)/$(BIN_WEB) ./cmd/anus-web

dev-ui:
	VITE_API=wails wails dev -tags $(TAGS)

dev-web:
	cd frontend && npm run dev &
	go run ./cmd/anus-web

frontend-wails:
	cd frontend && VITE_API=wails npm run build

frontend-web:
	cd frontend && VITE_API=web npm run build

test:
	go test ./...

docker-build: frontend-web
	docker build -t anus-web .

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
	install -Dm644 anus-ui.desktop $(PREFIX)/share/applications/anus-ui.desktop
	for sz in $(ICON_SIZES); do \
		mkdir -p $(PREFIX)/share/icons/hicolor/$${sz}x$${sz}/apps && \
		magick $(ICON_SRC) -resize $${sz}x$${sz} $(PREFIX)/share/icons/hicolor/$${sz}x$${sz}/apps/$(BIN_UI).png; \
	done

uninstall:
	rm -f $(PREFIX)/bin/$(BIN_UI)
	rm -f $(PREFIX)/share/applications/anus-ui.desktop
	for sz in $(ICON_SIZES); do \
		rm -f $(PREFIX)/share/icons/hicolor/$${sz}x$${sz}/apps/$(BIN_UI).png; \
	done

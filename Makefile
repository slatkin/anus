BIN    := anus
OUTDIR := build/bin

.PHONY: build frontend dev dev-docker docker-build test fmt vet tidy clean

build: frontend
	go build -tags production -o $(OUTDIR)/$(BIN) .

frontend:
	cd frontend && npm run build

dev:
	@trap 'kill 0' EXIT; \
	CONFIG=~/.config/anus/config.toml; \
	export MINIFLUX_API_KEY=$$(grep '^api_key' $$CONFIG | cut -d'"' -f2); \
	export MINIFLUX_URL=$$(grep '^server_url' $$CONFIG | cut -d'"' -f2); \
	export CACHE_DIR=$$HOME/.cache/anus-web; \
	cd frontend && npm run dev & go run .

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

docker-build: frontend
	docker buildx build -t anus-web .

test:
	go test ./...

fmt:
	gofmt -w .

vet:
	go vet ./...

tidy:
	go mod tidy

clean:
	rm -rf build frontend/dist

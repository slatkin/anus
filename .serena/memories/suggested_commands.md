# Suggested commands

## Development
```bash
make dev-ui        # Desktop hot-reload (Wails + VITE_API=wails); reads config from ~/.config/anus/config.toml
make dev-web       # Go API on :8080 + Vite dev server on :5173; reads config same file
make dev-docker    # Docker build + run on :8888
```

## Build
```bash
make build-ui      # Desktop binary via wails build -tags webkit2_41
make build-web     # Web binary: go build -tags production ./cmd/anus-web
```

## Testing
```bash
make test                                  # go test ./...
go test -run TestFetchEntries ./pkg/app    # single Go test
cd frontend && npm test                    # Vitest (frontend unit tests)
```

## Go utilities
```bash
make fmt    # gofmt -w .
make vet    # go vet ./...
make tidy   # go mod tidy
```

## Notes
- `dev-web` Makefile target reads API key + URL from config file automatically via grep.
- Desktop build requires `wails` CLI installed and GTK webkit2 libs.

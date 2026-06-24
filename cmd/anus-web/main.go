package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"

	"github.com/slatkin/anus/internal/cache"
	"github.com/slatkin/anus/pkg/app"
	"github.com/slatkin/anus/pkg/config"
	"github.com/slatkin/anus/pkg/miniflux"
)

// startupHooks are registered by frontend_dev.go or frontend_prod.go via init().
var startupHooks []func(*http.ServeMux)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config error: %v", err)
	}

	cacheDir := cfg.CacheDir
	if cacheDir == "" {
		cacheDir, err = cache.DefaultDir()
		if err != nil {
			log.Fatalf("cache dir error: %v", err)
		}
	}

	client := miniflux.NewClient(cfg.ServerUrl, cfg.ApiKey, cfg.AllowInvalidCerts)
	a := app.New(client, cfg.CacheExpiryDays)
	if err := a.Open(cacheDir); err != nil {
		log.Printf("Warning: %v (running without cache)", err)
	}
	defer a.Close()

	mux := http.NewServeMux()
	registerAPI(mux, a, cfg)
	for _, hook := range startupHooks {
		hook(mux)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	log.Printf("anus-web listening on :%s", port)
	log.Fatal(http.ListenAndServe(":"+port, mux))
}

func registerAPI(mux *http.ServeMux, a *app.App, cfg config.Config) {
	mux.HandleFunc("GET /api/cached", handleGetCached(a))
	mux.HandleFunc("GET /api/entries", handleGetEntries(a))
	mux.HandleFunc("POST /api/mark-read", handleMarkRead(a))
	mux.HandleFunc("POST /api/mark-unread", handleMarkUnread(a))
	mux.HandleFunc("POST /api/toggle-star", handleToggleStar(a))
	mux.HandleFunc("POST /api/save-entry", handleSaveEntry(a))
	mux.HandleFunc("POST /api/refresh-and-fetch", handleRefreshAndFetch(a))
	mux.HandleFunc("POST /api/clear-cache", handleClearCache(a))
	mux.HandleFunc("GET /api/search", handleSearch(a))
	mux.HandleFunc("GET /api/config", handleGetConfig(&cfg))
	mux.HandleFunc("POST /api/config", handlePostConfig(&cfg))
	mux.HandleFunc("GET /api/fetch-content", handleFetchContent(a))
}

// serveFrontend registers a handler that serves the embedded frontend dist.
// Called from the embed wrapper (embed.go) which is only compiled in production builds.
func serveFrontend(mux *http.ServeMux, distFS fs.FS) {
	sub, err := fs.Sub(distFS, "dist")
	if err != nil {
		panic(fmt.Sprintf("frontend embed error: %v", err))
	}
	fileServer := http.FileServer(http.FS(sub))
	mux.Handle("/", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Serve index.html for any non-asset path (SPA routing).
		if _, err := fs.Stat(sub, r.URL.Path[1:]); err != nil {
			r.URL.Path = "/"
		}
		fileServer.ServeHTTP(w, r)
	}))
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("json encode error: %v", err)
	}
}

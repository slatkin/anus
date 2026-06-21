package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"strconv"

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
	mux.HandleFunc("GET /api/cached", func(w http.ResponseWriter, r *http.Request) {
		result, err := a.FetchCached()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, result)
	})

	mux.HandleFunc("GET /api/entries", func(w http.ResponseWriter, r *http.Request) {
		result, err := a.FetchEntries()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, result)
	})

	mux.HandleFunc("POST /api/mark-read", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ IDs []int `json:"ids"` }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := a.MarkRead(body.IDs); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /api/mark-unread", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ IDs []int `json:"ids"` }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := a.MarkUnread(body.IDs); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /api/toggle-star", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ ID int `json:"id"` }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := a.ToggleStar(body.ID); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /api/save-entry", func(w http.ResponseWriter, r *http.Request) {
		var body struct{ ID int `json:"id"` }
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if err := a.SaveEntry(body.ID); err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("POST /api/refresh-and-fetch", func(w http.ResponseWriter, r *http.Request) {
		result, err := a.RefreshAndFetch()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, result)
	})

	mux.HandleFunc("POST /api/clear-cache", func(w http.ResponseWriter, r *http.Request) {
		result, err := a.ClearCache()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, result)
	})

	mux.HandleFunc("GET /api/search", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		result, err := a.SearchEntries(q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, result)
	})

	mux.HandleFunc("GET /api/config", func(w http.ResponseWriter, r *http.Request) {
		safe := cfg
		safe.ApiKey = ""
		writeJSON(w, safe)
	})

	mux.HandleFunc("POST /api/config", func(w http.ResponseWriter, r *http.Request) {
		var incoming config.Config
		if err := json.NewDecoder(r.Body).Decode(&incoming); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		path, err := config.GetConfigFilepath()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		if err := config.Save(incoming, path); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		cfg = incoming
		w.WriteHeader(http.StatusNoContent)
	})

	mux.HandleFunc("GET /api/fetch-content", func(w http.ResponseWriter, r *http.Request) {
		idStr := r.URL.Query().Get("id")
		url := r.URL.Query().Get("url")
		id, err := strconv.Atoi(idStr)
		if err != nil {
			http.Error(w, "invalid id", http.StatusBadRequest)
			return
		}
		html, err := a.FetchArticleContent(id, url)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, map[string]string{"content": html})
	})
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

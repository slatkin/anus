package main

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/slatkin/anus/pkg/app"
	"github.com/slatkin/anus/pkg/config"
)

func handleGetCached(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := a.FetchCached()
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		writeJSON(w, result)
	}
}

func handleGetEntries(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := a.FetchEntries()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, result)
	}
}

func handleMarkRead(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func handleMarkUnread(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func handleToggleStar(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func handleSaveEntry(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func handleRefreshAndFetch(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := a.RefreshAndFetch()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, result)
	}
}

func handleClearCache(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		result, err := a.ClearCache()
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, result)
	}
}

func handleSearch(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query().Get("q")
		result, err := a.SearchEntries(q)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadGateway)
			return
		}
		writeJSON(w, result)
	}
}

func handleGetConfig(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		safe := *cfg
		safe.ApiKey = ""
		writeJSON(w, safe)
	}
}

func handlePostConfig(cfg *config.Config) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
		*cfg = incoming
		w.WriteHeader(http.StatusNoContent)
	}
}

func handleFetchContent(a *app.App) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

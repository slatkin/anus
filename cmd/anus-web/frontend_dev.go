//go:build !production

package main

import (
	"log"
	"net/http"
)

func init() {
	startupHooks = append(startupHooks, func(mux *http.ServeMux) {
		log.Println("dev mode: frontend not embedded — run 'npm run dev' in frontend/")
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "dev mode: frontend served by Vite on port 5173", http.StatusNotFound)
		})
	})
}

//go:build production

package main

import (
	"net/http"

	"github.com/slatkin/anus/frontend"
)

func init() {
	startupHooks = append(startupHooks, func(mux *http.ServeMux) {
		serveFrontend(mux, frontend.FS)
	})
}

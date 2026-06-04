// Package frontend exposes the built Svelte app as an embedded filesystem.
// The dist/ directory must exist (run npm run build first).
package frontend

import "embed"

//go:embed all:dist
var FS embed.FS

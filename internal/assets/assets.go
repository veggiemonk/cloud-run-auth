// Package assets owns the embedded static file tree (CSS, images, JS)
// that every binary serves under /static/.
//
// Exists so the binaries are self-contained: a single `go build`
// produces an executable that needs no on-disk asset directory at
// runtime, which is exactly what Cloud Run's read-only container
// filesystem expects.
package assets

import "embed"

//go:embed static
var StaticFiles embed.FS

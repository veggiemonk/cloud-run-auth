// Package version owns the build identification the binaries log at
// startup: VCS revision, module version, build time, and the "dirty"
// flag from runtime/debug.ReadBuildInfo.
//
// Exists so no Makefile or release pipeline has to inject ldflags —
// Go's own build info is authoritative, and the binaries can be
// reproduced and identified from a plain `go build`.
//
// Adapted from https://github.com/imjasonh/version (see version.go).
package version

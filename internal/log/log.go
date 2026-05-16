// Package log builds a *slog.Logger from already-resolved options.
//
// Flag and env parsing live in internal/config — this package is
// deliberately small: it owns slog handler construction, level parsing,
// and nothing else.
//
// Conventions:
//   - Loggers write to stderr so stdout stays machine-readable.
//   - Text handler by default; json for structured consumers.
package log

import (
	"fmt"
	"io"
	"log/slog"
	"strings"
)

// Options configures a slog handler.
type Options struct {
	Format string // "text" or "json"
	Level  slog.Level
}

// ParseLevel accepts debug|info|warn|error (case-insensitive).
func ParseLevel(s string) (slog.Level, error) {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "debug":
		return slog.LevelDebug, nil
	case "info":
		return slog.LevelInfo, nil
	case "warn", "warning":
		return slog.LevelWarn, nil
	case "error", "err":
		return slog.LevelError, nil
	}
	return 0, fmt.Errorf("invalid log level %q (want debug|info|warn|error)", s)
}

// New returns a *slog.Logger that writes to w using opts.
func New(w io.Writer, opts Options) *slog.Logger {
	h := &slog.HandlerOptions{Level: opts.Level}
	if opts.Format == "json" {
		return slog.New(slog.NewJSONHandler(w, h))
	}
	return slog.New(slog.NewTextHandler(w, h))
}

// Discard returns a logger that discards all output; handy for tests.
func Discard() *slog.Logger {
	return slog.New(slog.DiscardHandler)
}

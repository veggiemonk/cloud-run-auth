package log_test

import (
	"bytes"
	"log/slog"
	"strings"
	"testing"

	"github.com/veggiemonk/cloud-run-auth/internal/is"
	"github.com/veggiemonk/cloud-run-auth/internal/log"
)

func TestParseLevel(t *testing.T) {
	cases := map[string]slog.Level{
		"debug": slog.LevelDebug,
		"INFO":  slog.LevelInfo,
		"Warn":  slog.LevelWarn,
		"err":   slog.LevelError,
	}
	for in, want := range cases {
		got, err := log.ParseLevel(in)
		is.NoErr(t, err)
		is.Equal(t, got, want, "")
	}
	_, err := log.ParseLevel("chatty")
	is.True(t, err != nil)
}

func TestNewJSONHandlerEmitsJSON(t *testing.T) {
	var buf bytes.Buffer
	lg := log.New(&buf, log.Options{Level: slog.LevelInfo, Format: "json"})
	lg.Info("hello", "key", "value")
	out := buf.String()
	is.True(t, strings.Contains(out, `"msg":"hello"`))
	is.True(t, strings.Contains(out, `"key":"value"`))
}

func TestNewTextHandlerRespectsLevel(t *testing.T) {
	var buf bytes.Buffer
	lg := log.New(&buf, log.Options{Level: slog.LevelWarn, Format: "text"})
	lg.Info("filtered")
	lg.Warn("kept")
	out := buf.String()
	is.True(t, !strings.Contains(out, "filtered"))
	is.True(t, strings.Contains(out, "kept"))
}

func TestDiscardLoggerIsSilent(t *testing.T) {
	lg := log.Discard()
	lg.Error("should vanish")
}

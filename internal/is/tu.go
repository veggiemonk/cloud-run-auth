// Package is provides testing utilities.
package is

import (
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func NoErr(tb testing.TB, err error) {
	tb.Helper()
	if err != nil {
		tb.Error(failure(Callers(), "", err.Error()))
	}
}

func True(tb testing.TB, b bool) {
	tb.Helper()
	if !b {
		tb.Error(failure(Callers(), "", "not true"))
	}
}

func Equal[T any](tb testing.TB, got, want T, message string, opts ...cmp.Option) {
	tb.Helper()
	if diff := Diff(got, want, opts...); diff != "" {
		tb.Error(failure(Callers(), message, diff))
	}
}

func failure(callers, message, body string) string {
	parts := make([]string, 0, 3)
	if callers != "" {
		parts = append(parts, callers)
	}
	if message != "" {
		parts = append(parts, message)
	}
	if body != "" {
		parts = append(parts, body)
	}

	return strings.Join(parts, "\n")
}

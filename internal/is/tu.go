// Package is owns the tiny in-repo test assertion vocabulary used by
// every *_test.go: NoErr / True / Equal, plus the Diff (go-cmp wrapper)
// and Callers (callsite stack) helpers behind their failure output.
//
// Exists to keep tests free of an external assertion dependency while
// still giving readable failures — failures print the callsite chain
// and a -got +want cmp.Diff, which is what we want, and nothing more.
// "A little copying is better than a little dependency."
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

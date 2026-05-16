package is

import (
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"strconv"
	"strings"

	"github.com/google/go-cmp/cmp"
)

// Diff compares two items and returns a human-readable diff string.
// If the items are equal, the string is empty.
func Diff[T any](got, want T, opts ...cmp.Option) string {
	// nolint: gocritic
	oo := append(opts, cmp.Exporter(func(reflect.Type) bool { return true }))

	diff := cmp.Diff(got, want, oo...)
	if diff != "" {
		return "\n-got +want\n" + diff
	}

	return ""
}

// Callers prints stack trace of callsites above assertion helpers.
func Callers() string {
	var pc [50]uintptr
	n := runtime.Callers(2, pc[:]) // skip runtime.Callers + Callers
	frames := runtime.CallersFrames(pc[:n])
	callsites := make([]string, 0, n)

	for {
		frame, more := frames.Next()
		if !skipFrame(frame) {
			callsites = append(callsites, frame.File+":"+strconv.Itoa(frame.Line))
		}
		if !more {
			break
		}
	}

	if len(callsites) == 0 {
		return ""
	}

	var b strings.Builder
	for _, v := range slices.Backward(callsites) {
		if b.Len() > 0 {
			b.WriteString(" -> ")
		}
		b.WriteString(filepath.Base(v))
	}

	return b.String() + ":"
}

func skipFrame(frame runtime.Frame) bool {
	if strings.HasPrefix(frame.Function, "runtime.") {
		return true
	}
	if strings.HasPrefix(frame.Function, "testing.") {
		return true
	}

	return false
}

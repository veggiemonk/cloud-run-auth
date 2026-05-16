// Package closeutil captures errors from deferred Close (or any `func() error`)
// into a named return, instead of silently dropping them.
//
// Usage:
//
//	func write(path string) (err error) {
//		f, err := os.Create(path)
//		if err != nil {
//			return err
//		}
//		defer closeutil.Do(&err, f.Close, "close %s", path)
//		...
//	}
//
// If Close returns an error and the function is already returning one, the
// Close error is joined onto it via errors.Join. If the function is returning
// nil, the Close error (wrapped with the formatted message) becomes the return.
//
// Inspired by github.com/efficientgo/core/errcapture. Kept local to avoid a
// new module dependency (North Star #1).
package closeutil

import (
	"errors"
	"fmt"
)

// Do captures the error returned by fn into *errp. Intended for `defer`.
//
// The format string/args are used only when fn returns an error; they describe
// what was being closed so the resulting error is diagnosable.
func Do(errp *error, fn func() error, format string, args ...any) {
	cerr := fn()
	if cerr == nil {
		return
	}
	wrapped := fmt.Errorf(format+": %w", append(args, cerr)...)
	if *errp == nil {
		*errp = wrapped
		return
	}
	*errp = errors.Join(*errp, wrapped)
}

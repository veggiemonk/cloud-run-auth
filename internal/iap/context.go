package iap

import (
	"context"
	"net/http"
)

type contextKey struct{}

// WithDetectionResult stores a DetectionResult in the request context.
func WithDetectionResult(r *http.Request, det DetectionResult) *http.Request {
	return r.WithContext(context.WithValue(r.Context(), contextKey{}, det))
}

// DetectionResultFromContext retrieves the DetectionResult from the request context.
// If no result is stored, it falls back to calling Detect(r).
func DetectionResultFromContext(r *http.Request) DetectionResult {
	if det, ok := r.Context().Value(contextKey{}).(DetectionResult); ok {
		return det
	}
	return Detect(r)
}

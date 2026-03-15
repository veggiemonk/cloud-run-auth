package iap

import (
	"net/http"
	"strings"
)

// DetectionResult holds the results of checking for IAP headers on a request.
type DetectionResult struct {
	HasJWT         bool   `json:"has_jwt"`
	HasEmailHeader bool   `json:"has_email_header"`
	HasIDHeader    bool   `json:"has_id_header"`
	Email          string `json:"email,omitempty"`
	UserID         string `json:"user_id,omitempty"`
	RawJWT         string `json:"-"`
	Warning        string `json:"warning,omitempty"`
}

// Detect checks the request for IAP-related headers and cross-validates them.
func Detect(r *http.Request) DetectionResult {
	result := DetectionResult{}

	rawJWT := r.Header.Get(HeaderJWTAssertion)
	if rawJWT != "" {
		result.HasJWT = true
		result.RawJWT = rawJWT
	}

	email := r.Header.Get(HeaderAuthenticatedEmail)
	if email != "" {
		result.HasEmailHeader = true
		// IAP prefixes email with "accounts.google.com:" — strip it.
		result.Email = strings.TrimPrefix(email, "accounts.google.com:")
	}

	userID := r.Header.Get(HeaderAuthenticatedID)
	if userID != "" {
		result.HasIDHeader = true
		result.UserID = strings.TrimPrefix(userID, "accounts.google.com:")
	}

	// Warn if unsigned headers are present without JWT — possible bypass.
	if !result.HasJWT && (result.HasEmailHeader || result.HasIDHeader) {
		result.Warning = "IAP email/ID headers present without JWT assertion. " +
			"These headers can be spoofed if IAP is not properly configured. " +
			"Always verify the JWT to confirm identity."
	}

	return result
}

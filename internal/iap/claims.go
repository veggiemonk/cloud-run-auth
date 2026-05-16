// Package iap owns everything the binaries need to consume Google
// Identity-Aware Proxy: the X-Goog-* header constants, the parsed Claims
// shape, header detection (Detect), JWT signature verification against
// Google's keys (Verifier), and the request-context plumbing that lets
// handlers read the detection result without re-parsing.
//
// Exists so the JWT-trust boundary lives in one place. Without
// verification, IAP's unsigned email/ID headers are spoofable by any
// caller that can reach the service — so this package centralises the
// "is this request really from IAP" decision and surfaces a warning
// whenever unsigned headers appear without a JWT.
package iap

import "time"

const (
	// HeaderJWTAssertion is the IAP JWT assertion header.
	HeaderJWTAssertion = "X-Goog-IAP-JWT-Assertion"
	// HeaderAuthenticatedEmail is the IAP authenticated user email header.
	HeaderAuthenticatedEmail = "X-Goog-Authenticated-User-Email"
	// HeaderAuthenticatedID is the IAP authenticated user ID header.
	HeaderAuthenticatedID = "X-Goog-Authenticated-User-ID"
)

// Claims represents the parsed claims from an IAP JWT.
type Claims struct {
	Issuer       string    `json:"iss"`
	Subject      string    `json:"sub"`
	Email        string    `json:"email"`
	HostedDomain string    `json:"hd,omitempty"`
	Audience     string    `json:"aud"`
	IssuedAt     time.Time `json:"iat"`
	ExpiresAt    time.Time `json:"exp"`
	AccessLevels []string  `json:"access_levels,omitempty"`
}

// ClaimDescriptions maps JWT claim names to human-readable descriptions.
var ClaimDescriptions = map[string]string{
	"iss":           "Issuer — who created and signed the token",
	"sub":           "Subject — unique identifier for the authenticated user",
	"email":         "Email — the user's email address",
	"hd":            "Hosted Domain — the Google Workspace domain of the user",
	"aud":           "Audience — intended recipient of the token (your service)",
	"iat":           "Issued At — when the token was created",
	"exp":           "Expires At — when the token expires",
	"access_levels": "Access Levels — context-aware access levels granted",
}

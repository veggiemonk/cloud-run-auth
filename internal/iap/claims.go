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

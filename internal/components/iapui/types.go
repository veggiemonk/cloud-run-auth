// Package iapui owns the templ view models and generated page renderers
// for the runiap diagnostic UI: dashboard, headers, JWT inspector,
// audience checker, request log, and diagnostic check pages, plus the
// IAPNav shared with the layout chrome in internal/shared/components.
//
// Exists to keep render-only concerns (struct shapes the templates
// consume, IAP-app nav layout) out of the iaphandler package — handlers
// build a *Data struct from internal/iap and hand it to the templ page;
// the view layer never reaches back into the auth model.
package iapui

import (
	"github.com/veggiemonk/cloud-run-auth/internal/iap"
	shared "github.com/veggiemonk/cloud-run-auth/internal/shared/components"
	"github.com/veggiemonk/cloud-run-auth/internal/shared/reqlog"
)

// DashboardData holds the data for the dashboard view.
type DashboardData struct {
	Email        string `json:"email,omitempty"`
	UserID       string `json:"user_id,omitempty"`
	HostedDomain string `json:"hosted_domain,omitempty"`
	IAPWarning   string `json:"iap_warning,omitempty"`
	JWTError     string `json:"jwt_error,omitempty"`
	HasIAP       bool   `json:"has_iap"`
	JWTValid     bool   `json:"jwt_valid"`
}

// HeaderEntry represents a single HTTP header for display.
type HeaderEntry struct {
	Name  string `json:"name"`
	Value string `json:"value"`
	IsIAP bool   `json:"is_iap"`
}

// HeadersData holds the data for the headers view.
type HeadersData struct {
	Headers []HeaderEntry `json:"headers"`
}

// JWTData holds the data for the JWT inspection view.
// RawJWT is intentionally excluded to prevent bearer token leakage via JSON API.
type JWTData struct {
	Claims          *iap.Claims `json:"claims,omitempty"`
	HeaderJSON      string      `json:"header_json,omitempty"`
	PayloadJSON     string      `json:"payload_json,omitempty"`
	SignatureB64    string      `json:"signature_b64,omitempty"`
	ValidationError string      `json:"validation_error,omitempty"`
	Present         bool        `json:"present"`
	Valid           bool        `json:"valid"`
}

// AudienceData holds the data for the audience validation view.
type AudienceData struct {
	CurrentAudience  string `json:"current_audience,omitempty"`
	ExpectedAudience string `json:"expected_audience,omitempty"`
	FormatHelp       string `json:"format_help"`
	Match            bool   `json:"match"`
	Checked          bool   `json:"checked"`
}

// Check represents a single diagnostic check result.
type Check struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "pass", "fail", "warn"
	Detail string `json:"detail"`
}

// DiagnosticData holds the data for the diagnostic view.
type DiagnosticData struct {
	Checks []Check `json:"checks"`
}

// LogData holds the data for the request log view.
type LogData struct {
	Entries []reqlog.Entry `json:"entries"`
}

// IAPNav returns the NavConfig for the IAP application.
func IAPNav() shared.NavConfig {
	return shared.NavConfig{
		Brand: "RunIAP",
		Items: []shared.NavItem{
			{Href: "/", Label: "Dashboard", Page: "dashboard"},
			{Href: "/headers", Label: "Headers", Page: "headers"},
			{Href: "/jwt", Label: "JWT", Page: "jwt"},
			{Href: "/audience", Label: "Audience", Page: "audience"},
			{Href: "/log", Label: "Log", Page: "log"},
			{Href: "/diagnostic", Label: "Diagnostic", Page: "diagnostic"},
		},
	}
}

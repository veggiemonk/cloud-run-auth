// Package components owns the templ layout primitives shared by every
// per-binary UI bundle (iapui, oauthui): the page Layout, NavConfig, and
// NavItem types that each binary parameterises with its own brand and
// nav links.
//
// Exists so the dashboard chrome is defined once. Per-binary view
// packages keep their own content templates but render through this
// Layout — adding a route in iapui or oauthui requires no change here.
package components

// NavItem represents a single navigation link.
type NavItem struct {
	Href  string
	Label string
	Page  string
}

// NavConfig holds navigation configuration for the layout.
type NavConfig struct {
	Brand string
	Items []NavItem
}

// Package config owns the env-driven runtime configuration for every binary
// under cmd/ (runiap, runoauth, runoauthprod).
//
// Each binary has a typed Load*() entry point that returns:
//   - a populated struct (with defaults, env overrides, ardanlabs/conf flag
//     parsing for --help),
//   - a help string (non-empty iff --help was requested),
//   - an error.
//
// All three configs embed Common, which carries the fields every server
// needs (Port, LogLevel, LogFormat) and exposes LogOptions() for plugging
// into internal/log. Secrets are tagged `mask` so String() can be logged at
// startup without leaking credentials.
//
// 12-factor by design: no config file, no XDG lookup — Cloud Run injects
// env vars, locally a `.env` or shell export is enough.
package config

import (
	"errors"
	"fmt"

	"github.com/ardanlabs/conf/v3"

	"github.com/veggiemonk/cloud-run-auth/internal/log"
)

// Common is the configuration shared by every binary: serving port and
// logger knobs. Embedded in each per-binary struct.
type Common struct {
	Port      string `conf:"env:PORT,default:8080"`
	LogLevel  string `conf:"env:LOG_LEVEL,default:info"`
	LogFormat string `conf:"env:LOG_FORMAT,default:json"`
}

// LogOptions returns log.Options derived from the parsed log fields.
// Panics if LogLevel is unparseable — Validate is expected to have run.
func (c Common) LogOptions() log.Options {
	lvl, err := log.ParseLevel(c.LogLevel)
	if err != nil {
		panic(fmt.Sprintf("config: log level %q unvalidated: %v", c.LogLevel, err))
	}
	return log.Options{Level: lvl, Format: c.LogFormat}
}

// Validate enforces invariants on the shared fields.
func (c Common) Validate() error {
	if _, err := log.ParseLevel(c.LogLevel); err != nil {
		return fmt.Errorf("LOG_LEVEL: %w", err)
	}
	switch c.LogFormat {
	case "text", "json":
	default:
		return fmt.Errorf("LOG_FORMAT: %q (want text|json)", c.LogFormat)
	}
	return nil
}

// IAP is the configuration for the runiap binary. JWT signature verification
// is enabled when Audience is non-empty; absent audience on Cloud Run
// (KService set) is a misconfiguration that the binary logs at error.
type IAP struct {
	Common

	Audience string `conf:"env:IAP_AUDIENCE"`
	KService string `conf:"env:K_SERVICE"`
}

// OAuth is the configuration for the runoauth binary — the dev/local OAuth
// flow backed by the in-memory session store.
type OAuth struct {
	Common

	ClientID     string `conf:"env:GOOGLE_CLIENT_ID,required"`
	ClientSecret string `conf:"env:GOOGLE_CLIENT_SECRET,required,mask"`
	RedirectURL  string `conf:"env:OAUTH_REDIRECT_URL,default:http://localhost:8080/auth/callback"`
}

// OAuthProd is the configuration for the runoauthprod binary — production
// OAuth with Firestore-backed sessions, CSRF, rate limiting, and
// allowed-domain gating.
type OAuthProd struct {
	Common

	Google struct {
		OAuthConfig  string `conf:"env:GOOGLE_OAUTH_CONFIG,mask"`
		ClientID     string `conf:"env:GOOGLE_CLIENT_ID"`
		ClientSecret string `conf:"env:GOOGLE_CLIENT_SECRET,mask"`
		RedirectURL  string `conf:"env:OAUTH_REDIRECT_URL,default:http://localhost:8080/auth/callback"`
	}
	ProjectID            string `conf:"env:PROJECT_ID,required"`
	FirestoreDB          string `conf:"env:FIRESTORE_DATABASE,default:(default)"`
	SessionEncryptionKey string `conf:"env:SESSION_ENCRYPTION_KEY,required,mask"`
	CSRFKey              string `conf:"env:CSRF_KEY,mask"`
	KRevision            string `conf:"env:K_REVISION"`
	AllowedDomain        string `conf:"env:ALLOWED_DOMAIN,default:myowndomain.com"`
	AuthRateLimitBurst   int    `conf:"env:AUTH_RATE_LIMIT_BURST,default:20"`
	UserRateLimitBurst   int    `conf:"env:USER_RATE_LIMIT_BURST,default:60"`
}

// LoadIAP parses IAP configuration from the environment.
// Returns (cfg, helpText, err). When helpText is non-empty the caller
// should print it and exit 0 — --help was requested.
func LoadIAP() (IAP, string, error) {
	var cfg IAP
	help, err := conf.Parse("", &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			return cfg, help, nil
		}
		return cfg, "", fmt.Errorf("parsing config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, "", err
	}
	return cfg, "", nil
}

// LoadOAuth parses dev OAuth configuration from the environment.
func LoadOAuth() (OAuth, string, error) {
	var cfg OAuth
	help, err := conf.Parse("", &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			return cfg, help, nil
		}
		return cfg, "", fmt.Errorf("parsing config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, "", err
	}
	return cfg, "", nil
}

// LoadOAuthProd parses production OAuth configuration from the environment.
func LoadOAuthProd() (OAuthProd, string, error) {
	var cfg OAuthProd
	help, err := conf.Parse("", &cfg)
	if err != nil {
		if errors.Is(err, conf.ErrHelpWanted) {
			return cfg, help, nil
		}
		return cfg, "", fmt.Errorf("parsing config: %w", err)
	}
	if err := cfg.Validate(); err != nil {
		return cfg, "", err
	}
	return cfg, "", nil
}

// String returns a redacted dump of cfg suitable for startup logging.
// Fields tagged `mask` are replaced with xxxxxx by ardanlabs/conf.
func String(cfg any) (string, error) { return conf.String(cfg) }

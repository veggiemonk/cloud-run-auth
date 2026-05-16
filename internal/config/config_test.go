package config_test

import (
	"os"
	"strings"
	"testing"

	"github.com/veggiemonk/cloud-run-auth/internal/config"
	"github.com/veggiemonk/cloud-run-auth/internal/is"
)

// clearEnv unsets every env var the loaders look at so ambient developer
// shells don't leak into expectations. Cleanup restores prior values.
// Note: `t.Setenv(k, "")` sets the var to empty (still present) which
// ardanlabs/conf treats as a real value — bypassing defaults and tripping
// `required`. We need true unset.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"PORT", "LOG_LEVEL", "LOG_FORMAT",
		"IAP_AUDIENCE", "K_SERVICE",
		"GOOGLE_CLIENT_ID", "GOOGLE_CLIENT_SECRET", "OAUTH_REDIRECT_URL",
		"GOOGLE_OAUTH_CONFIG",
		"PROJECT_ID", "FIRESTORE_DATABASE", "SESSION_ENCRYPTION_KEY", "CSRF_KEY",
		"K_REVISION", "ALLOWED_DOMAIN",
		"AUTH_RATE_LIMIT_BURST", "USER_RATE_LIMIT_BURST",
	} {
		// Seed t.Setenv so its cleanup restores the original on test exit,
		// then immediately Unsetenv to make the var absent during the test.
		if orig, ok := os.LookupEnv(k); ok {
			t.Setenv(k, orig)
		}
		os.Unsetenv(k)
	}
}

// silenceArgs swaps os.Args to argv[:1] for the duration of the test.
// ardanlabs/conf reads os.Args by default; under `go test` it would
// otherwise try to parse -test.* flags and fail.
func silenceArgs(t *testing.T) {
	t.Helper()
	orig := os.Args
	os.Args = os.Args[:1]
	t.Cleanup(func() { os.Args = orig })
}

func TestLoadIAPDefaults(t *testing.T) {
	clearEnv(t)
	silenceArgs(t)
	cfg, help, err := config.LoadIAP()
	is.NoErr(t, err)
	is.Equal(t, help, "", "no help requested")
	is.Equal(t, cfg.Port, "8080", "")
	is.Equal(t, cfg.LogLevel, "info", "")
	is.Equal(t, cfg.LogFormat, "json", "")
	is.Equal(t, cfg.Audience, "", "")
}

func TestLoadIAPEnvOverride(t *testing.T) {
	clearEnv(t)
	silenceArgs(t)
	t.Setenv("PORT", "9090")
	t.Setenv("IAP_AUDIENCE", "/projects/123/global/backendServices/abc")
	t.Setenv("K_SERVICE", "myservice")
	cfg, _, err := config.LoadIAP()
	is.NoErr(t, err)
	is.Equal(t, cfg.Port, "9090", "")
	is.Equal(t, cfg.Audience, "/projects/123/global/backendServices/abc", "")
	is.Equal(t, cfg.KService, "myservice", "")
}

func TestLoadOAuthRequiresClientCreds(t *testing.T) {
	clearEnv(t)
	silenceArgs(t)
	_, _, err := config.LoadOAuth()
	is.True(t, err != nil)
}

func TestLoadOAuthHappyPath(t *testing.T) {
	clearEnv(t)
	silenceArgs(t)
	t.Setenv("GOOGLE_CLIENT_ID", "id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "secret")
	cfg, _, err := config.LoadOAuth()
	is.NoErr(t, err)
	is.Equal(t, cfg.ClientID, "id", "")
	is.Equal(t, cfg.ClientSecret, "secret", "")
	is.Equal(t, cfg.RedirectURL, "http://localhost:8080/auth/callback", "default applied")
}

func TestLoadOAuthProdRequiresSecrets(t *testing.T) {
	clearEnv(t)
	silenceArgs(t)
	_, _, err := config.LoadOAuthProd()
	is.True(t, err != nil)
}

func TestLoadOAuthProdHappyPath(t *testing.T) {
	clearEnv(t)
	silenceArgs(t)
	t.Setenv("PROJECT_ID", "p")
	t.Setenv("SESSION_ENCRYPTION_KEY", "base64key=")
	cfg, _, err := config.LoadOAuthProd()
	is.NoErr(t, err)
	is.Equal(t, cfg.ProjectID, "p", "")
	is.Equal(t, cfg.SessionEncryptionKey, "base64key=", "")
	is.Equal(t, cfg.AllowedDomain, "myowndomain.com", "default applied")
	is.Equal(t, cfg.AuthRateLimitBurst, 20, "default applied")
	is.Equal(t, cfg.UserRateLimitBurst, 60, "default applied")
	is.Equal(t, cfg.FirestoreDB, "(default)", "default applied")
}

func TestStringMasksSecrets(t *testing.T) {
	clearEnv(t)
	silenceArgs(t)
	t.Setenv("GOOGLE_CLIENT_ID", "id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "thesecret")
	cfg, _, err := config.LoadOAuth()
	is.NoErr(t, err)
	s, err := config.String(&cfg)
	is.NoErr(t, err)
	is.True(t, !strings.Contains(s, "thesecret"))
}

func TestBadLogLevelRejected(t *testing.T) {
	clearEnv(t)
	silenceArgs(t)
	t.Setenv("GOOGLE_CLIENT_ID", "id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "s")
	t.Setenv("LOG_LEVEL", "chatty")
	_, _, err := config.LoadOAuth()
	is.True(t, err != nil)
}

func TestBadLogFormatRejected(t *testing.T) {
	clearEnv(t)
	silenceArgs(t)
	t.Setenv("GOOGLE_CLIENT_ID", "id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "s")
	t.Setenv("LOG_FORMAT", "yaml")
	_, _, err := config.LoadOAuth()
	is.True(t, err != nil)
}

func TestLogOptionsRoundtrip(t *testing.T) {
	clearEnv(t)
	silenceArgs(t)
	t.Setenv("GOOGLE_CLIENT_ID", "id")
	t.Setenv("GOOGLE_CLIENT_SECRET", "s")
	t.Setenv("LOG_LEVEL", "debug")
	t.Setenv("LOG_FORMAT", "text")
	cfg, _, err := config.LoadOAuth()
	is.NoErr(t, err)
	opts := cfg.LogOptions()
	is.Equal(t, opts.Format, "text", "")
}

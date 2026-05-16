# TODO

> **Purpose:** ledger for deferred work — things that cannot be completed now (out of scope, blocked, needs decision, deferred improvement). Never silently drop incomplete work; append here.
>
> **Format:** `- [priority] <title> — <one-line context>`. Priorities: P0 (urgent), P1 (soon), P2 (eventually). Top entry is what to do next.

## Open

- [P1] golangci-lint findings from new `.golangci.yml` — 26 issues surfaced by adopting the template's linter config (run `go tool mage lint` to reproduce). Breakdown:
  - **gosec G114** (×2) — `cmd/runiap/main.go`, `cmd/runoauth/main.go`: `http.ListenAndServe` without timeouts. Decide: harden the demo servers (use `http.Server` with read/header/idle timeouts like `cmd/runoauthprod`) or document why demos skip it.
  - **gosec G124** (×3) — `cmd/runoauthprod/cookies.go:29`, `internal/oauth/google.go:76,136`: cookies missing `Secure`/`HttpOnly`/`SameSite` attributes. Audit each and tighten.
  - **noctx** (×12) — 10 in test files (`httptest.NewRequest` → `httptest.NewRequestWithContext(t.Context(), …)`), 2 production callsites in `internal/oauth/google.go:154` and `cmd/runoauthprod/auth.go:255` (`client.Get` → `client.Do(NewRequestWithContext(...))` against Google's userinfo endpoint).
  - **forcetypeassert** (×5) — `cmd/runoauthprod/auth.go:214`, `internal/middleware/ratelimit.go:44,48,87,91`: replace `x.(T)` with the comma-ok form and handle the fail path.
  - **gocritic** (×3) — `cmd/runoauthprod/main.go:54` (`exitAfterDefer`: `os.Exit` skips the `store.Close` defer — refactor to return-from-run-func), plus `if/else` chains in `internal/handler/{iap,oauth}handler/diagnostic.go` to convert to `switch`.
  - **prealloc** (×1) — `internal/handler/iaphandler/headers.go:16`: preallocate `entries` with `len(r.Header)`.
  - Once cleared, `go tool mage check` will be green and CI (`.github/workflows/ci.yml`) will pass.

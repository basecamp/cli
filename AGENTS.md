# Development context for basecamp/cli

Shared Go toolkit for 37signals CLI development. This is a library repo — no binary output.

## Repo structure

```
output/          Structured JSON envelopes, exit codes, TTY formatting
credstore/       Credential storage (keyring + file fallback)
pkce/            PKCE code verifier/challenge (RFC 7636)
oauthcallback/   Local HTTP server for OAuth callbacks
profile/         Named environment profiles
surface/         CLI surface snapshots and compatibility diffing

seed/            Templates for bootstrapping new CLIs
actions/         Reusable GitHub Actions (rubric-check, surface-compat, sync-skills)
skills/          Agent skills distributed via basecamp/skills
prompts/         Agent prompts (seed-cli.md, close-gap.md)

RUBRIC.md        37signals CLI rubric specification
Makefile         Build and test targets
```

## Packages

All packages import from `github.com/basecamp/cli/<package>`.

| Package | Purpose |
|---------|---------|
| `output` | JSON response/error envelopes, 8 typed exit codes (0–8), TTY auto-detection |
| `credstore` | System keyring with file fallback (0600); caller-configured `DisableEnvVar` forces file mode |
| `pkce` | `GenerateVerifier()` and `GenerateChallenge()` for OAuth PKCE flows |
| `oauthcallback` | `WaitForCallback()` starts local server, returns authorization code |
| `profile` | Named profiles (`--profile`, `APP_PROFILE`), base URL + app-specific settings |
| `surface` | `Snapshot()` walks Cobra tree; `Diff()` detects breaking removals |

## Testing

```
make check       # fmt-check + vet + test — the inner-loop default
make test-race   # go test -race ./...
make lint        # golangci-lint run
make check-all   # full CI suite
```

## Seed templates

Templates in `seed/` use Go text/template syntax. `.tmpl` files are processed; all others copy verbatim. Template variables include app name, API base URL, module path, and auth model.

When authoring new seed templates:
- Use `.tmpl` extension only for files needing variable substitution
- Keep generated code minimal — point to shared packages where possible
- Test by running the `prompts/seed-cli.md` prompt end-to-end

## Rubric

[RUBRIC.md](RUBRIC.md) defines the quality standard for 37signals Go CLIs. Two profiles:
- **API CLI** — full-featured product CLI (all 4 tiers)
- **TUI tool** — single-purpose terminal tool (subset of tiers)

The `actions/rubric-check` action automates scoring against a built binary.

## Code style

- `gofmt` formatting (enforced by `make fmt-check`)
- Follow [Effective Go](https://go.dev/doc/effective_go)
- Tests live alongside source (`*_test.go` in same package)

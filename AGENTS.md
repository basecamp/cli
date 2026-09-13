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
prompts/         Agent prompts (seed-cli.md, close-gap.md, close-input-gap.md)

RUBRIC.md        37signals CLI rubric specification
Makefile         Build and test targets
```

## Packages

All packages import from `github.com/basecamp/cli/<package>`.

| Package | Purpose |
|---------|---------|
| `output` | JSON response/error envelopes, typed exit codes (see `output/codes.go`), TTY auto-detection |
| `credstore` | System keyring with file fallback (0600); caller-configured `DisableEnvVar` forces file mode |
| `pkce` | `GenerateVerifier()` and `GenerateChallenge()` for OAuth PKCE flows |
| `oauthcallback` | `WaitForCallback()` starts local server, returns authorization code |
| `profile` | Named profiles (`--profile`, `APP_PROFILE`), base URL + app-specific settings |
| `surface` | `Snapshot()` walks Cobra tree; `Diff()` detects breaking removals |
| `editor` | `Open(initialContent)` launches `$EDITOR`, returns edited text |

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

## Skills sync

Every CLI publishes its `skills/<name>/` trees into `skills/<name>/` at the root of
`basecamp/skills` (the layout `npx skills add basecamp/skills` reads). The one
implementation is `seed/scripts/sync-skills.sh`; `scripts/sync-skills.sh` here execs it
with `SYNC_SOURCE=cli`, and `actions/sync-skills` runs it from the action's checkout.
Change the seed script, never a copy.

Several CLIs share that target, so each one owns `.managed-skills.<source>` there
(`<source>` is the publishing repo: `hey-cli`, `basecamp-cli`, `cli`) and removes only
skill directories its own manifest lists, that its skill set no longer has, and that no
other manifest claims. The legacy shared `.managed-skills` is rewritten as a comment-only
tombstone so a sibling still on the pre-fix script deletes nothing (basecamp/skills#5).
The script always clones the target fresh (from `SKILLS_REPO_URL`, default
`https://github.com/basecamp/skills.git`) and pushes only the commit it made, so there is
no checkout to hand it. `seed/scripts/test-sync-skills.sh` runs the script as two CLIs
against a local bare repository — real clones, commits and pushes, no network — and is
part of `make check`.

## Rubric

[RUBRIC.md](RUBRIC.md) defines the quality standard for 37signals Go CLIs. Two profiles:
- **API CLI** — full-featured product CLI (all 4 tiers)
- **TUI tool** — single-purpose terminal tool (subset of tiers)

The `actions/rubric-check` action automates scoring against a built binary.

## Code style

- `gofmt` formatting (enforced by `make fmt-check`)
- Follow [Effective Go](https://go.dev/doc/effective_go)
- Tests live alongside source (`*_test.go` in same package)

# Releasing

## Quick release

```bash
make release VERSION=0.1.0
```

## Dry run

```bash
make release VERSION=0.1.0 DRY_RUN=1
```

## What happens

1. Validates semver format, main branch, clean tree, synced with remote
2. Runs `make release-check` (fmt, vet, lint, test, race, bench, tidy-check, vuln, secrets)
3. Creates annotated tag `v$VERSION` and pushes to origin
4. GitHub Actions [release workflow](.github/workflows/release.yml) runs:
   - Security scan (gitleaks, trivy, gosec, CodeQL, dependency-review)
   - Full test suite with race detector + govulncheck
   - Skills sync to `basecamp/skills` (when configured)
5. Tagged Go module is published via the [module proxy](https://proxy.golang.org) automatically

This is a library — no binary builds, no GoReleaser, no platform distribution.

## Versioning

Pre-1.0: minor bumps for features, patch bumps for fixes.

Consumers import packages as `github.com/basecamp/cli/<package>` and pin
via `go get github.com/basecamp/cli@v0.x.y`.

## Requirements

- On `main` branch with clean, synced working tree
- `make release-check` passes
- Go toolchain matches `.mise.toml` (currently Go 1.26)

### Toolchain reset

If you see `toolchain mismatch` or stale stdlib cache errors:

```bash
mise install          # sync Go version from .mise.toml
go clean -cache       # clear build cache
go vet ./...          # verify clean build
```

## CI secrets

| Secret/Variable | Purpose | Required |
|----------------|---------|----------|
| `SKILLS_APP_ID` (var) | GitHub App ID for skills sync bot | Optional |
| `SKILLS_APP_PRIVATE_KEY` (secret) | GitHub App private key for skills sync | Optional |

Skills sync is disabled by default. Configure both `SKILLS_APP_ID` and
`SKILLS_APP_PRIVATE_KEY` to enable automatic skill distribution on release.

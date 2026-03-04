# 37signals CLI Rubric

A specification that codifies the design decisions from `basecamp-cli` into a reusable standard for all 37signals Go CLIs. Use this rubric to evaluate existing CLIs, guide new ones, and ensure consistency across the portfolio.

## Profiles

| Profile | Scope | Tiers |
|---------|-------|-------|
| **API CLI** | Full-featured CLI wrapping a 37signals product API (e.g. `basecamp`, `hey`) | All 4 tiers, all criteria |
| **TUI tool** | Single-purpose terminal UI or developer tool | 1D (Auth), 4A (Distribution), 4B (Testing), 4D (Dev Experience), plus TUI-specific criteria where noted |

Criteria marked **(API)** apply only to the API CLI profile. All other criteria apply to both profiles.

---

## Philosophy

### 1. Structured output is the default
Every command returns a JSON envelope (`{ok, data, summary, breadcrumbs}`) when piped or when `--json` is passed. TTY gets styled output. The same command serves humans and machines.

### 2. Typed exit codes are a contract
Eight codes (0–8) map to categories agents can branch on without parsing stderr. `0=OK, 1=Usage, 2=NotFound, 3=Auth, 4=Forbidden, 5=RateLimit, 6=Network, 7=API, 8=Ambiguous`.

### 3. Programmatic discovery beats documentation
`--help --agent` returns structured JSON. Breadcrumbs in every response suggest next actions. `commands --json` returns the full catalog. An agent can explore the CLI without reading docs.

### 4. Breadcrumbs are navigation, not decoration
Every success response includes suggested follow-up commands. This is the primary mechanism for agent chaining — each response tells the agent what to do next.

### 5. Errors are actionable data
Every error carries a machine-readable code, a human hint, and a retryable flag. Agents retry rate limits automatically. Humans see what to do.

### 6. TTY auto-detection makes the right thing easy
`FormatAuto` resolves based on stdout. No flags needed for the common case. `--json` forces JSON. `--styled` forces ANSI. The CLI adapts to its context.

### 7. Surface stability enables automation
Flag and subcommand removals are breaking changes caught by CI. Agents can depend on the CLI's surface as a stable API.

---

## Tier 1: Agent Contract

### 1A. Structured Output **(API)**

| # | Criterion | Seed | pkg | Reference |
|---|-----------|------|-----|-----------|
| 1A.1 | `--json` flag on every command | Yes | `output` | `internal/output/format.go` |
| 1A.2 | TTY auto-detection: styled on terminal, JSON when piped | Yes | `output` | `internal/output/format.go` |
| 1A.3 | Success envelope: `{ok: true, data, summary, breadcrumbs}` | Yes | `output` | `internal/output/envelope.go` |
| 1A.4 | Error envelope: `{ok: false, error, code, hint}` | Yes | `output` | `internal/output/errors.go` |
| 1A.5 | `--quiet` flag: raw JSON data, no envelope | Yes | `output` | `internal/output/format.go` |
| 1A.6 | `--agent` flag: quiet JSON + suppress interactive prompts | Yes | `output` | `internal/output/format.go` |
| 1A.7 | `--ids-only`: one ID per line | No | `output` | `internal/output/format.go` |
| 1A.8 | `--count`: integer count only | No | `output` | `internal/output/format.go` |
| 1A.9 | `--markdown`: literal GFM output | No | — | — |
| 1A.10 | Large integer ID preservation (`json.Decoder.UseNumber`) | Yes | `output` | `internal/output/json.go` |

### 1B. Exit Codes

| # | Criterion | Seed | pkg | Reference |
|---|-----------|------|-----|-----------|
| 1B.1 | 8 typed exit codes: 0=OK 1=Usage 2=NotFound 3=Auth 4=Forbidden 5=RateLimit 6=Network 7=API 8=Ambiguous | Yes | `output` | `internal/output/exit.go` |
| 1B.2 | Machine-readable code strings in error envelope **(API)** | Yes | `output` | `internal/output/errors.go` |
| 1B.3 | Typed error constructors | Yes | `output` | `internal/output/errors.go` |
| 1B.4 | Errors carry retryable flag | Yes | `output` | `internal/output/errors.go` |
| 1B.5 | Errors carry actionable hint text | Yes | `output` | `internal/output/errors.go` |

### 1C. Programmatic Discovery **(API)**

| # | Criterion | Seed | pkg | Reference |
|---|-----------|------|-----|-----------|
| 1C.1 | `--help --agent` emits structured JSON | Yes | `surface` | `internal/cli/help.go` |
| 1C.2 | Breadcrumbs in every success response | Yes | `output` | `internal/output/envelope.go` |
| 1C.3 | `<cli> commands --json` returns full catalog | No | — | `internal/commands/commands.go` |
| 1C.4 | Agent notes via command annotations | No | — | `internal/commands/commands.go` |
| 1C.5 | Interactive prompts suppressed for machine output | Yes | `output` | `internal/output/format.go` |

### 1D. Authentication

| # | Criterion | Seed | pkg | Reference |
|---|-----------|------|-----|-----------|
| 1D.1 | `APP_TOKEN` env var bypasses all interactive auth | Yes | — | `internal/auth/token.go` |
| 1D.2 | Interactive auth flow (OAuth+PKCE, token wizard, etc.) | Yes | `pkce`, `oauthcallback` | `internal/auth/oauth.go` |
| 1D.3 | System keyring preferred, file fallback (0600) | Yes | `credstore` | `internal/auth/credstore/` |
| 1D.4 | Token auto-refresh with expiry buffer | Yes | — | `internal/auth/refresh.go` |
| 1D.5 | Auth management commands (login, logout, status) | Yes | — | `internal/commands/auth.go` |
| 1D.6 | `APP_NO_KEYRING=1` env to force file storage | Yes | `credstore` | `internal/auth/credstore/` |

---

## Tier 2: Reliability

### 2A. Surface Stability

| # | Criterion | Seed | pkg | Reference |
|---|-----------|------|-----|-----------|
| 2A.1 | `--version` with embedded version, commit, date | Yes | — | `internal/version/version.go` |
| 2A.2 | CLI surface snapshot generation | No | `surface` | `internal/surface/` |
| 2A.3 | Surface compat check in CI (fail on removals) | No | `surface` | `.github/workflows/` |
| 2A.4 | Command catalog parity test **(API)** | No | — | `internal/commands/commands_test.go` |
| 2A.5 | Cobra error messages normalized | No | — | `internal/cli/root.go` |

### 2B. Resilience

| # | Criterion | Seed | pkg | Reference |
|---|-----------|------|-----|-----------|
| 2B.1 | Retry with exponential backoff | Yes | — | `internal/resilience/retry.go` |
| 2B.2 | Rate limit handling (429, Retry-After) | Yes | `output` | `internal/resilience/ratelimit.go` |
| 2B.3 | Circuit breaker | No | — | — |
| 2B.4 | Request concurrency limiter | No | — | — |

### 2C. Configuration

| # | Criterion | Seed | pkg | Reference |
|---|-----------|------|-----|-----------|
| 2C.1 | Layered config: flag > env > local > repo > global > default | Yes | — | `internal/config/` |
| 2C.2 | Source tracking on every config value | Yes | — | `internal/config/source.go` |
| 2C.3 | `config show` with source attribution **(API)** | Yes | — | `internal/commands/config.go` |
| 2C.4 | Per-repo config at git root | Yes | — | `internal/config/` |
| 2C.5 | HTTPS enforcement for non-localhost | Yes | — | `internal/config/` |
| 2C.6 | XDG directory compliance (config/state/cache separation) | Yes | — | `internal/config/` |
| 2C.7 | Named profiles (`--profile`, `APP_PROFILE`, default_profile) | Yes | `profile` | `internal/config/` |

---

## Tier 3: Agent Integration **(API)**

### 3A. Skill & Plugin

| # | Criterion | Seed | pkg | Reference |
|---|-----------|------|-----|-----------|
| 3A.1 | `SKILL.md` embedded via `go:embed` | Yes | — | `skills/SKILL.md` |
| 3A.2 | `<cli> skill` prints embedded skill | Yes | — | `internal/commands/skill.go` |
| 3A.3 | `.claude-plugin/` with plugin.json, hooks, agents | Yes | — | `.claude-plugin/` |
| 3A.4 | SessionStart hook emits CLI context | Yes | — | `.claude-plugin/hooks/` |
| 3A.5 | Skill synced to `basecamp/skills` on release | Yes | — | `scripts/sync-skills.sh` |

### 3B. Pagination

| # | Criterion | Seed | pkg | Reference |
|---|-----------|------|-----|-----------|
| 3B.1 | `--limit N` to cap results | Yes | — | `internal/commands/` |
| 3B.2 | `--all` to fetch all pages | Yes | — | `internal/commands/` |
| 3B.3 | Truncation notice in response | Yes | `output` | `internal/output/envelope.go` |

### 3C. Observability

| # | Criterion | Seed | pkg | Reference |
|---|-----------|------|-----|-----------|
| 3C.1 | `--verbose` / `-v` (stackable) | Yes | — | `internal/cli/root.go` |
| 3C.2 | `APP_DEBUG` env var | Yes | — | `internal/cli/root.go` |
| 3C.3 | `--stats` adds `meta.stats` to envelope **(API)** | No | — | — |

---

## Tier 4: Distribution & Ecosystem

### 4A. Distribution

| # | Criterion | Seed | pkg | Reference |
|---|-----------|------|-----|-----------|
| 4A.1 | Cross-platform builds (darwin/linux arm64+amd64, windows amd64) | Yes | — | `.goreleaser.yml` |
| 4A.2 | Homebrew tap | Yes | — | `.goreleaser.yml` |
| 4A.3 | One-line install script | Yes | — | `install.sh` |
| 4A.4 | GoReleaser (or equivalent) release automation | Yes | — | `.goreleaser.yml` |
| 4A.5 | SHA256 checksums + cosign signing | Yes | — | `.goreleaser.yml` |
| 4A.6 | SBOM generation | Yes | — | `.goreleaser.yml` |
| 4A.7 | macOS notarization | Yes | — | `.goreleaser.yml` |
| 4A.8 | Scoop (Windows) | Yes | — | `.goreleaser.yml` |
| 4A.9 | AUR (Arch Linux) | Yes | — | `scripts/publish-aur.sh` |

### 4B. Testing

| # | Criterion | Seed | pkg | Reference |
|---|-----------|------|-----|-----------|
| 4B.1 | Unit tests | Yes | — | `*_test.go` |
| 4B.2 | E2E integration tests (BATS or subprocess) | Yes | — | `e2e/` |
| 4B.3 | E2E in CI | Yes | — | `.github/workflows/` |
| 4B.4 | TUI integration tests (appDriver pattern, if TUI exists) | No | — | — |
| 4B.5 | Race detection in CI | No | — | — |
| 4B.6 | Fuzz testing for parsers | No | — | — |
| 4B.7 | Benchmarks with regression detection | No | — | — |

### 4C. Shell Completion **(API)**

| # | Criterion | Seed | pkg | Reference |
|---|-----------|------|-----|-----------|
| 4C.1 | `<cli> completion bash/zsh/fish/powershell` | Yes | — | `internal/commands/completion.go` |
| 4C.2 | File-based completion cache with TTL | No | — | — |
| 4C.3 | Flag-specific dynamic completions | No | — | — |

### 4D. Developer Experience

| # | Criterion | Seed | pkg | Reference |
|---|-----------|------|-----|-----------|
| 4D.1 | README with quick-start, examples, output format docs | Yes | — | `README.md` |
| 4D.2 | CONTRIBUTING.md | Yes | — | `CONTRIBUTING.md` |
| 4D.3 | AGENTS.md with coding style section | Yes | — | `AGENTS.md` |
| 4D.4 | Makefile with build, test, test-e2e, check, lint | Yes | — | `Makefile` |
| 4D.5 | golangci-lint with committed config | Yes | — | `.golangci.yml` |
| 4D.6 | `doctor` command (connectivity, auth, config, cache, completion) | Yes | — | `internal/commands/doctor.go` |
| 4D.7 | `setup` command with first-run auto-detection | Yes | — | `internal/commands/wizard.go` |
| 4D.8 | API coverage tracking **(API)** | Yes | — | `API-COVERAGE.md` |
| 4D.9 | CI pipeline (test, lint, security, e2e, surface) | Yes | — | `.github/workflows/` |

---

## Scoring Template

Copy this template and fill it in to score a CLI against the rubric.

For the **TUI tool profile**, score only the applicable tiers (1D, 4A, 4B, 4D) and mark all **(API)** criteria as N/A.

```markdown
## Scorecard: [CLI Name]

| Tier | Score | Max |
|------|-------|-----|
| T1: Agent Contract | /26 | 26 |
| T2: Reliability | /16 | 16 |
| T3: Agent Integration | /11 | 11 |
| T4: Distribution | /28 | 28 |
| **Total** | **/81** | **81** |

### Detailed Results

| # | Criterion | Pass | N/A | Notes |
|---|-----------|------|-----|-------|
| 1A.1 | `--json` flag on every command | | | |
| 1A.2 | TTY auto-detection | | | |
| 1A.3 | Success envelope | | | |
| 1A.4 | Error envelope | | | |
| 1A.5 | `--quiet` flag | | | |
| 1A.6 | `--agent` flag | | | |
| 1A.7 | `--ids-only` | | | |
| 1A.8 | `--count` | | | |
| 1A.9 | `--markdown` | | | |
| 1A.10 | Large integer ID preservation | | | |
| 1B.1 | 8 typed exit codes | | | |
| 1B.2 | Machine-readable code strings | | | |
| 1B.3 | Typed error constructors | | | |
| 1B.4 | Retryable flag | | | |
| 1B.5 | Actionable hint text | | | |
| 1C.1 | `--help --agent` | | | |
| 1C.2 | Breadcrumbs | | | |
| 1C.3 | `commands --json` | | | |
| 1C.4 | Agent notes | | | |
| 1C.5 | Prompts suppressed | | | |
| 1D.1 | Token env var | | | |
| 1D.2 | Interactive auth flow | | | |
| 1D.3 | System keyring + file fallback | | | |
| 1D.4 | Token auto-refresh | | | |
| 1D.5 | Auth management commands | | | |
| 1D.6 | No-keyring env var | | | |
| 2A.1 | `--version` | | | |
| 2A.2 | Surface snapshot | | | |
| 2A.3 | Surface compat CI | | | |
| 2A.4 | Catalog parity test | | | |
| 2A.5 | Cobra errors normalized | | | |
| 2B.1 | Retry with backoff | | | |
| 2B.2 | Rate limit handling | | | |
| 2B.3 | Circuit breaker | | | |
| 2B.4 | Concurrency limiter | | | |
| 2C.1 | Layered config | | | |
| 2C.2 | Source tracking | | | |
| 2C.3 | `config show` | | | |
| 2C.4 | Per-repo config | | | |
| 2C.5 | HTTPS enforcement | | | |
| 2C.6 | XDG compliance | | | |
| 2C.7 | Named profiles | | | |
| 3A.1 | Embedded SKILL.md | | | |
| 3A.2 | `skill` command | | | |
| 3A.3 | `.claude-plugin/` | | | |
| 3A.4 | SessionStart hook | | | |
| 3A.5 | Skill synced on release | | | |
| 3B.1 | `--limit N` | | | |
| 3B.2 | `--all` | | | |
| 3B.3 | Truncation notice | | | |
| 3C.1 | `--verbose` / `-v` | | | |
| 3C.2 | Debug env var | | | |
| 3C.3 | `--stats` | | | |
| 4A.1 | Cross-platform builds | | | |
| 4A.2 | Homebrew tap | | | |
| 4A.3 | Install script | | | |
| 4A.4 | GoReleaser | | | |
| 4A.5 | Checksums + signing | | | |
| 4A.6 | SBOM | | | |
| 4A.7 | macOS notarization | | | |
| 4A.8 | Scoop (Windows) | | | |
| 4A.9 | AUR (Arch Linux) | | | |
| 4B.1 | Unit tests | | | |
| 4B.2 | E2E tests | | | |
| 4B.3 | E2E in CI | | | |
| 4B.4 | TUI integration tests | | | |
| 4B.5 | Race detection | | | |
| 4B.6 | Fuzz testing | | | |
| 4B.7 | Benchmarks | | | |
| 4C.1 | Shell completion | | | |
| 4C.2 | Completion cache | | | |
| 4C.3 | Dynamic completions | | | |
| 4D.1 | README | | | |
| 4D.2 | CONTRIBUTING.md | | | |
| 4D.3 | AGENTS.md | | | |
| 4D.4 | Makefile | | | |
| 4D.5 | golangci-lint | | | |
| 4D.6 | `doctor` command | | | |
| 4D.7 | `setup` command | | | |
| 4D.8 | API coverage tracking | | | |
| 4D.9 | CI pipeline | | | |
```

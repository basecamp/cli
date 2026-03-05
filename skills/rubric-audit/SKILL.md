---
name: rubric-audit
description: Audit a Go CLI against the 37signals CLI rubric
---

# CLI Rubric Audit

Audit a Go CLI repository against the 37signals CLI rubric (RUBRIC.md).

## Usage

Run this skill in the root of a Go CLI repository to produce a gap report.

## Audit Process

### 1. Identify the CLI

- Find the main binary (check `cmd/` directory or Makefile)
- Build it: `make build` or `go build ./cmd/<name>`
- Determine the profile: API CLI (wraps a web API) or TUI tool (full-screen interface)

### 2. Check Tier 1: Agent Contract

#### 1A. Structured Output (API CLI only)
- [ ] Run `<cli> --help` — does `--json` flag exist?
- [ ] Pipe a command: `<cli> <cmd> | cat` — does it output JSON automatically?
- [ ] Run with `--json`: verify `{ok: true, data: ...}` envelope
- [ ] Run invalid command: verify `{ok: false, error: ..., code: ...}` envelope
- [ ] Check for `--quiet`, `--agent`, `--ids-only`, `--count`, `--markdown` flags
- [ ] Grep for `json.Decoder.UseNumber` or `json.Number` in output code

#### 1B. Exit Codes
- [ ] Run with bad args: should exit 1
- [ ] Access nonexistent resource: should exit 2
- [ ] Run without auth: should exit 3
- [ ] Check error types in code: look for typed error constructors

#### 1C. Programmatic Discovery (API CLI only)
- [ ] Run `--help --agent`: should emit structured JSON
- [ ] Check responses for breadcrumbs
- [ ] Look for `commands --json` or catalog command

#### 1D. Authentication
- [ ] Check for `APP_TOKEN` env var support
- [ ] Check for keyring usage (go-keyring dependency)
- [ ] Check for file fallback with 0600 perms
- [ ] Check for token refresh logic

### 3. Check Tier 2: Reliability

#### 2A. Surface Stability
- [ ] `--version` flag exists and shows version/commit/date
- [ ] Surface snapshot script or tool exists
- [ ] CI runs surface compat check

#### 2B. Resilience
- [ ] Grep for retry/backoff logic
- [ ] Check for 429/rate limit handling

#### 2C. Configuration
- [ ] Check config loading order (flag > env > file)
- [ ] Check for HTTPS enforcement
- [ ] Check for XDG directory usage

### 4. Check Tier 3: Agent Integration (API CLI only)

- [ ] Check for SKILL.md and go:embed
- [ ] Check for .claude-plugin/ directory
- [ ] Check for --limit, --all flags on list commands
- [ ] Check for --verbose, APP_DEBUG

#### 3A.6: setup claude
- [ ] Check for `internal/harness/` directory with Claude detection code
- [ ] Run `<cli> setup claude --help` — subcommand exists
- [ ] Grep for `ClaudeMarketplaceSource` or `basecamp/claude-plugins` in harness code

#### 3A.7: Marketplace registration (manual, external)
- [ ] This criterion requires checking the external `basecamp/claude-plugins` repo
- [ ] If not locally available, flag as "cannot verify — check basecamp/claude-plugins manually"
- [ ] If available: verify `.claude-plugin/marketplace.json` contains an entry with `"name": "<cli>"`

### 5. Check Tier 4: Distribution & Ecosystem

- [ ] Check for .goreleaser.yaml
- [ ] Check for Homebrew tap
- [ ] Check for e2e tests
- [ ] Check for golangci-lint config
- [ ] Check for CONTRIBUTING.md, AGENTS.md

## Output Format

Produce a scorecard:

```
## Scorecard: <CLI Name>

| Tier | Score | Max |
|------|-------|-----|
| T1: Agent Contract | X/26 | 26 |
| T2: Reliability | X/16 | 16 |
| T3: Agent Integration | X/13 | 13 |
| T4: Distribution | X/29 | 29 |
| **Total** | **X/84** | **84** |

### Critical Gaps
1. [Most impactful gap]
2. [Second gap]
...

### Recommended Priority
1. [First thing to fix — highest leverage]
2. [Second]
...
```

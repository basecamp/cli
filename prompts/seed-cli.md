# Bootstrap a New CLI from Seed

You are creating a new Go CLI for a 37signals product using the seed templates.

## Input

- **App name**: e.g., "fizzy", "hey"
- **API base URL**: e.g., "https://fizzy.37signals.com"
- **Auth model**: OAuth+PKCE, bearer token (PAT), or purchase token

## Process

1. Create the repository structure:
   ```
   <app>-cli/
   ├── cmd/<app>/main.go
   ├── internal/
   │   ├── auth/
   │   ├── commands/
   │   ├── config/
   │   ├── harness/
   │   └── output/
   ├── e2e/
   ├── skills/
   ├── scripts/
   ├── .claude-plugin/
   ├── .github/
   │   ├── workflows/
   │   │   ├── test.yml
   │   │   ├── security.yml
   │   │   ├── release.yml
   │   │   ├── ai-labeler.yml
   │   │   ├── dependabot-auto-merge.yml
   │   │   └── labeler.yml
   │   ├── prompts/
   │   │   ├── classify-pr.prompt.yml
   │   │   ├── detect-breaking.prompt.yml
   │   │   └── summarize-changelog.prompt.yml
   │   ├── codeql/
   │   │   └── codeql-config.yml
   │   ├── CODEOWNERS
   │   ├── dependabot.yml
   │   ├── labeler.yml
   │   ├── pull_request_template.md
   │   └── release.yml
   ├── go.mod
   ├── Makefile
   ├── .goreleaser.yaml
   ├── .golangci.yml
   ├── .gitleaks.toml
   ├── .pre-commit-config.yaml
   ├── AGENTS.md
   ├── CONTRIBUTING.md
   ├── RELEASING.md
   └── README.md
   ```

2. Initialize go.mod with `github.com/basecamp/<app>-cli`

3. Add shared dependencies:
   ```
   go get github.com/basecamp/cli/output
   go get github.com/basecamp/cli/credstore
   go get github.com/basecamp/cli/pkce
   go get github.com/spf13/cobra
   ```

4. Copy and customize seed templates:

   **Build & lint:**
   - `seed/Makefile` → `Makefile` (update BINARY_NAME, LEGACY_PATTERN)
   - `seed/.goreleaser.yaml` → `.goreleaser.yaml` (update ProjectName)
   - `seed/.golangci.yml` → `.golangci.yml`

   **Docs & project config:**
   - `seed/AGENTS.md.tmpl` → `AGENTS.md` (fill in app name)
   - `seed/CONTRIBUTING.md.tmpl` → `CONTRIBUTING.md` (fill in app name)
   - `seed/API-COVERAGE.md.tmpl` → `API-COVERAGE.md` (fill in app name)
   - `seed/RELEASING.md.tmpl` → `RELEASING.md` (fill in app name, org, repo)

   **Source code:**
   - `seed/internal/output/output.go` → `internal/output/output.go`
   - `seed/internal/auth/auth.go` → `internal/auth/auth.go` (customize service name, env vars)
   - `seed/internal/commands/doctor.go.tmpl` → `internal/commands/doctor.go`
   - `seed/internal/commands/setup.go.tmpl` → `internal/commands/setup.go`
   - `seed/internal/commands/skill.go.tmpl` → `internal/commands/skill.go`
   - `seed/internal/harness/claude.go.tmpl` → `internal/harness/claude.go` (fill in app name)

   **Skills & plugin:**
   - `seed/.claude-plugin/` → `.claude-plugin/` (customize)
   - `mkdir -p .claude-plugin/skills && ln -s ../../skills/<app> .claude-plugin/skills/<app>` (create skills symlink)
   - `seed/skills/app/SKILL.md.tmpl` → `skills/<app>/SKILL.md` (customize)
   - `seed/skills/embed.go.tmpl` → `skills/embed.go`

   **Scripts:**
   - `seed/scripts/release.sh.tmpl` → `scripts/release.sh` (fill in org, repo; chmod +x)
   - `seed/scripts/check-cli-surface.sh` → `scripts/check-cli-surface.sh` (copy; chmod +x)
   - `seed/scripts/check-cli-surface-diff.sh` → `scripts/check-cli-surface-diff.sh` (copy; chmod +x)
   - `seed/scripts/collect-profile.sh` → `scripts/collect-profile.sh` (copy; chmod +x)
   - `seed/scripts/publish-aur.sh` → `scripts/publish-aur.sh` (copy; chmod +x)
   - `seed/scripts/sync-skills.sh` → `scripts/sync-skills.sh` (copy; chmod +x)

   **GitHub infra (copy as-is unless .tmpl):**
   - `seed/.github/workflows/test.yml` → `.github/workflows/test.yml` (update env vars, GOPRIVATE)
   - `seed/.github/workflows/security.yml` → `.github/workflows/security.yml`
   - `seed/.github/workflows/release.yml` → `.github/workflows/release.yml` (update env vars)
   - `seed/.github/workflows/ai-labeler.yml` → `.github/workflows/ai-labeler.yml`
   - `seed/.github/workflows/dependabot-auto-merge.yml` → `.github/workflows/dependabot-auto-merge.yml`
   - `seed/.github/workflows/labeler.yml` → `.github/workflows/labeler.yml`
   - `seed/.github/dependabot.yml` → `.github/dependabot.yml`
   - `seed/.github/CODEOWNERS.tmpl` → `.github/CODEOWNERS` (fill in team name)
   - `seed/.github/pull_request_template.md` → `.github/pull_request_template.md`
   - `seed/.github/release.yml` → `.github/release.yml`
   - `seed/.github/labeler.yml.tmpl` → `.github/labeler.yml` (customize label rules)
   - `seed/.github/codeql/codeql-config.yml` → `.github/codeql/codeql-config.yml`
   - `seed/.github/prompts/classify-pr.prompt.yml` → `.github/prompts/classify-pr.prompt.yml`
   - `seed/.github/prompts/detect-breaking.prompt.yml` → `.github/prompts/detect-breaking.prompt.yml`
   - `seed/.github/prompts/summarize-changelog.prompt.yml` → `.github/prompts/summarize-changelog.prompt.yml`

   **Local dev config:**
   - `seed/.pre-commit-config.yaml.tmpl` → `.pre-commit-config.yaml` (fill in env var name)
   - `seed/.gitleaks.toml.tmpl` → `.gitleaks.toml` (customize allowlist)

5. Create the root command in `cmd/<app>/main.go`:
   - Import `github.com/spf13/cobra`
   - Add persistent flags: --json, --quiet, --agent, --verbose, --ids-only, --count, --markdown
   - Wire up output.Writer with format resolution
   - Add --help --agent handler

6. Create auth commands: `<app> auth login`, `<app> auth logout`, `<app> auth status`

7. Create first resource command as an example

8. Run `make check` to verify everything works

## Post-bootstrap: GitHub infra setup

After the repo is pushed to GitHub:

1. **Required labels** — create `bug`, `enhancement`, `documentation`, `breaking` labels
   (the AI labeler and release changelog reference them)

2. **Branch protection** — protect `main` with required status checks from test.yml

3. **Secrets & vars** — configure optional features per the matrix in `RELEASING.md`:

   | Feature | What to configure |
   |---------|-------------------|
   | Private module access | `vars.RELEASE_CLIENT_ID` + `secrets.RELEASE_APP_PRIVATE_KEY` |
   | AI changelog | `vars.ENABLE_AI_CHANGELOG=true` |
   | macOS notarization | 5 secrets in `release` environment |
   | Homebrew tap | `secrets.HOMEBREW_TAP_TOKEN` |
   | AUR publish | `secrets.AUR_SSH_KEY` |
   | Skills sync | `vars.SKILLS_APP_ID` + `secrets.SKILLS_APP_PRIVATE_KEY` |

   All features are off by default and degrade gracefully.

4. **Pre-commit hooks** — install locally:
   ```
   pip install pre-commit && pre-commit install --install-hooks
   ```

5. **Claude plugin marketplace** — register in `basecamp/claude-plugins`:
   - Clone `basecamp/claude-plugins`
   - Add entry to `.claude-plugin/marketplace.json` plugins array:
     `{"name": "<app>", "description": "...", "source": {"source": "github", "repo": "basecamp/<app>-cli"}, "category": "productivity"}`
   - PR and merge

## Auth Model Configuration

### OAuth + PKCE
```go
import (
    "github.com/basecamp/cli/credstore"
    "github.com/basecamp/cli/pkce"
    "github.com/basecamp/cli/oauthcallback"
)
```

### Bearer Token (PAT)
```go
import "github.com/basecamp/cli/credstore"
// No PKCE or callback needed — user provides token directly
```

### Purchase Token (HMAC)
```go
import "github.com/basecamp/cli/credstore"
// Custom verification logic — credstore handles storage only
```

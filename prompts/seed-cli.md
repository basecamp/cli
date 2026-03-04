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
   │   └── output/
   ├── e2e/
   ├── skills/
   ├── .claude-plugin/
   ├── go.mod
   ├── Makefile
   ├── .goreleaser.yaml
   ├── .golangci.yml
   ├── AGENTS.md
   ├── CONTRIBUTING.md
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
   - `seed/Makefile` → `Makefile` (update BINARY_NAME)
   - `seed/.goreleaser.yaml` → `.goreleaser.yaml` (update ProjectName)
   - `seed/.golangci.yml` → `.golangci.yml`
   - `seed/AGENTS.md.tmpl` → `AGENTS.md` (fill in app name)
   - `seed/CONTRIBUTING.md.tmpl` → `CONTRIBUTING.md` (fill in app name)
   - `seed/internal/output/output.go` → `internal/output/output.go`
   - `seed/internal/auth/auth.go` → `internal/auth/auth.go` (customize service name, env vars)
   - `seed/.claude-plugin/` → `.claude-plugin/` (customize)
   - `seed/skills/SKILL.md.tmpl` → `skills/SKILL.md` (customize)

5. Create the root command in `cmd/<app>/main.go`:
   - Import `github.com/spf13/cobra`
   - Add persistent flags: --json, --quiet, --agent, --verbose, --ids-only, --count, --markdown
   - Wire up output.Writer with format resolution
   - Add --help --agent handler

6. Create auth commands: `<app> auth login`, `<app> auth logout`, `<app> auth status`

7. Create first resource command as an example

8. Run `make check` to verify everything works

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

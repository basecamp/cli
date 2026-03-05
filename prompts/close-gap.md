# Close a Rubric Gap

You are closing a specific gap in a Go CLI's compliance with the 37signals CLI rubric.

## Input

- **Criterion ID**: e.g., "1A.1" (--json flag on every command)
- **CLI repo**: The repository you're working in
- **Current state**: What exists today

## Process

1. Read RUBRIC.md to understand the criterion requirements
2. Read the reference implementation in basecamp-cli
3. Assess the current state of the target CLI
4. Implement the minimum change to meet the criterion
5. Add or update tests
6. Verify the criterion is met

## Shared Packages

If the gap can be closed by adopting a shared package, prefer that over writing new code:

| Gap Area | Package | Import |
|----------|---------|--------|
| Output envelope, formats, exit codes | `output` | `github.com/basecamp/cli/output` |
| Credential storage (keyring + file) | `credstore` | `github.com/basecamp/cli/credstore` |
| PKCE helpers | `pkce` | `github.com/basecamp/cli/pkce` |
| OAuth callback server | `oauthcallback` | `github.com/basecamp/cli/oauthcallback` |
| CLI surface snapshots | `surface` | `github.com/basecamp/cli/surface` |

## Implementation Patterns

### Adding --json flag (1A.1)
Add a persistent `--json` flag to the root command. In your output wrapper, check the flag and set `FormatJSON` accordingly.

### Adding structured output (1A.3-4)
Import `github.com/basecamp/cli/output` and use `Writer.OK()` / `Writer.Err()` in every command's RunE.

### Adding exit codes (1B.1)
Use `output.AsError(err).ExitCode()` in your root command's error handler. Map all errors through typed constructors.

### Adding --help --agent (1C.1)
Detect `--help --agent` flag combination. When both are set, emit a JSON object with: name, description, flags (name, type, default, description), subcommands (name, description).

### Adding keyring (1D.3)
Replace file-only credential storage with `credstore.NewStore()`. Set ServiceName to your app name, DisableEnvVar to `APP_NO_KEYRING`.

### Adding surface stability (2A.2-3)
Use the `surface` package to generate snapshots. Commit the baseline. Add the `surface-compat` GitHub Action to CI.

### Adding setup claude (3A.6)
Create `internal/harness/claude.go` with `ClaudeMarketplaceSource` and `ClaudePluginName`
constants, plus `DetectClaude`, `FindClaudeBinary`, `IsPluginNeeded`, and `CheckClaudePlugin`
functions. Add a `setup claude` subcommand that runs marketplace add (best-effort) then
plugin install, with verify-after-install. Wire into the main setup wizard and add
breadcrumb suggestions via `harness.IsPluginNeeded()`.
Reference: github.com/basecamp/basecamp-cli/internal/harness/claude.go and wizard.go.

### Marketplace registration (3A.7)
Manual, external follow-up. Add a plugin entry to `basecamp/claude-plugins`
marketplace.json with source pointing at `basecamp/<app>-cli`. This is a one-time
step in the marketplace repo, not automatable from within the CLI repo.

# cli

Shared Go toolkit for building 37signals product CLIs.

Provides reusable packages, seed templates, GitHub Actions, and a rubric specification that standardize CLI development across 37signals products (Basecamp, HEY, Fizzy, etc.).

## Packages

| Package | Import | Description |
|---------|--------|-------------|
| `output` | `github.com/basecamp/cli/output` | Structured JSON envelopes, typed exit codes, and TTY-aware formatting |
| `credstore` | `github.com/basecamp/cli/credstore` | Credential storage with system keyring preference and file fallback (0600) |
| `pkce` | `github.com/basecamp/cli/pkce` | PKCE code verifier and challenge generation (RFC 7636) |
| `oauthcallback` | `github.com/basecamp/cli/oauthcallback` | Local HTTP server that captures OAuth authorization code callbacks |
| `profile` | `github.com/basecamp/cli/profile` | Named environment profiles for targeting different accounts or environments |
| `surface` | `github.com/basecamp/cli/surface` | CLI surface snapshot and compatibility diffing for Cobra command trees |

## Seed templates

The `seed/` directory contains templates for bootstrapping a new 37signals Go CLI. Templates include project scaffolding for auth, commands, output formatting, distribution, skills, and CI.

Use the `prompts/seed-cli.md` agent prompt to generate a new CLI from these templates:

```
Inputs: app name, API base URL, auth model (OAuth+PKCE, bearer token, or purchase token)
```

## GitHub Actions

Reusable composite actions in `actions/`:

| Action | Description |
|--------|-------------|
| `rubric-check` | Score a built CLI binary against the 37signals CLI rubric |
| `surface-compat` | Fail CI if CLI flags or subcommands were removed (breaking change) |
| `sync-skills` | Sync embedded SKILL.md files to the `basecamp/skills` distribution repo on release |

Usage in a workflow:

```yaml
- uses: basecamp/cli/actions/rubric-check@main
  with:
    cli-binary: ./dist/myapp
```

## Rubric

[RUBRIC.md](RUBRIC.md) codifies design decisions from `basecamp-cli` into a reusable standard covering:

- **Tier 1** — Agent contract: structured output, exit codes, discovery, auth
- **Tier 2** — Reliability: surface stability, resilience, configuration
- **Tier 3** — Agent integration: skills, pagination, observability
- **Tier 4** — Distribution & ecosystem: builds, testing, shell completion, DX

## Agent prompts

Reusable agent prompts in `prompts/`:

| Prompt | Purpose |
|--------|---------|
| `seed-cli.md` | Bootstrap a new CLI from the seed templates |
| `close-gap.md` | Close a specific rubric gap in an existing CLI |

## Skills

The `skills/` directory contains agent skills distributed via `basecamp/skills`:

- `rubric-audit` — Audit a Go CLI against the rubric

## Development

Requires Go 1.24+.

```
make check       # fmt-check + vet + test (inner-loop dev)
make test         # go test ./...
make test-race    # go test -race ./...
make lint         # golangci-lint run
make check-all    # full suite: fmt-check + vet + lint + test-race + bench
```

## License

Copyright 2025 37signals LLC. Released under the [MIT License](MIT-LICENSE).

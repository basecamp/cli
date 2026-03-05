# Contributing to basecamp/cli

## Setup

```
git clone https://github.com/basecamp/cli.git
cd cli
```

Requires **Go 1.24+**.

## Running checks

```
make check       # fmt-check + vet + test (fast, run before every commit)
make test        # go test ./...
make test-race   # go test -race ./...
make lint        # golangci-lint run (install: https://golangci-lint.run)
make check-all   # full CI suite: fmt-check + vet + lint + test-race + bench
```

## Pull requests

1. Run `make check` before pushing.
2. Add tests for new behavior.
3. Keep commits focused — one logical change per commit.
4. Open a PR against `main`.

## Code style

- `gofmt` formatting — enforced by CI.
- Follow [Effective Go](https://go.dev/doc/effective_go).
- Tests live alongside source (`*_test.go` in the same package).

## Project structure

This is a Go library with no binary output. Key areas:

- **Packages** (`output/`, `credstore/`, `pkce/`, `oauthcallback/`, `profile/`, `surface/`) — reusable libraries imported by product CLIs
- **Seed templates** (`seed/`) — project scaffolding for new CLIs
- **GitHub Actions** (`actions/`) — composite actions for CI
- **Rubric** (`RUBRIC.md`) — quality specification for 37signals CLIs

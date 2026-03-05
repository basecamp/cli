# Makefile Convention

Standard Makefile targets for 37signals Go CLIs and libraries. The seed template (`seed/Makefile`) is the canonical implementation. The `rubric-check` action verifies conformance.

## Targets

### Required (all repos)

| Target | Default? | Contract |
|--------|----------|----------|
| `check` | **Yes** | Fast checks for inner-loop dev. Must be the first target so bare `make` runs it. |
| `test` | | `go test ./...` — unit tests only, no network, no external deps. |
| `vet` | | `go vet ./...` — static analysis. |
| `fmt` | | `gofmt -w .` — fix formatting in place. |
| `fmt-check` | | Fail with exit 1 if any file needs formatting. Print offending files. |

### Required (CLI repos with a binary)

| Target | Contract |
|--------|----------|
| `build` | `go build -o ./bin/BINARY ./cmd/BINARY` — deterministic output path. |
| `test-e2e` | `bats e2e/` — end-to-end integration tests. |
| `clean` | `rm -rf ./bin` — remove build artifacts. |

### Optional (recommended)

| Target | Contract |
|--------|----------|
| `lint` | `golangci-lint run` — requires golangci-lint installed. |
| `test-race` | `go test -race ./...` — tests with race detector. |
| `bench` | `go test -bench=. -benchmem ./...` — benchmarks. |
| `bench-cpu` | Benchmarks with CPU profile output. |
| `bench-mem` | Benchmarks with memory profile output. |
| `check-toolchain` | Guard against Go toolchain mismatch (PATH go vs GOROOT go). Wired as prereq to `build` and `test`. |
| `test-coverage` | `go test -coverprofile=coverage.out` + generate `coverage.html`. |
| `coverage` | Alias for `test-coverage` that auto-opens the report in a browser. |
| `check-all` | Full CI suite: fmt-check + vet + lint + test-race + test-e2e + bench. |

## Composition Rules

- **`check`** must be the **first target** (Make's default). Bare `make` should always work.
- **`check`** is fast. It runs what you'd run before every commit: `fmt-check vet test` (libraries) or `fmt-check vet test test-e2e` (CLIs).
- **`check-all`** is thorough. It runs what CI runs: adds lint, race detection, and benchmarks. Slower, but catches more.
- **`lint`** is separate from `check` because golangci-lint is an external install. `check` should work with just Go.
- Targets must not swallow errors. Every command should fail the target on non-zero exit.

## Variables

CLI Makefiles should define at the top:

```makefile
BINARY_NAME := $(shell basename $(CURDIR))
BUILD_DIR := ./bin
```

This derives the binary name from the directory (e.g., `basecamp-cli/` → `basecamp-cli`). Override `BINARY_NAME` if the binary should differ from the directory name.

## Library repos

Library repos (like `basecamp/cli` itself) omit `build`, `test-e2e`, and `clean` since there's no binary. Their `check` is: `fmt-check vet test`.

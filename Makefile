.DEFAULT_GOAL := check

.PHONY: check test test-race vet lint fmt fmt-check bench check-all \
       tidy tidy-check replace-check vuln secrets security release-check release

# Default target: fast checks for inner-loop dev.
check: fmt-check vet test

test:
	go test ./...

test-race:
	go test -race ./...

vet:
	go vet ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Run 'make fmt' to fix formatting" && gofmt -l . && exit 1)

bench:
	go test -bench=. -benchmem ./...

# Tidy dependencies
tidy:
	go mod tidy

# Verify go.mod/go.sum are tidy (CI gate)
tidy-check:
	@set -e; cp go.mod go.mod.tidycheck; cp go.sum go.sum.tidycheck; \
	restore() { mv go.mod.tidycheck go.mod; mv go.sum.tidycheck go.sum; }; \
	if ! go mod tidy; then \
		restore; \
		echo "'go mod tidy' failed. Restored original go.mod/go.sum."; \
		exit 1; \
	fi; \
	if ! git diff --quiet -- go.mod go.sum; then \
		restore; \
		echo "go.mod/go.sum are not tidy. Run 'make tidy' and commit the result."; \
		exit 1; \
	fi; \
	rm -f go.mod.tidycheck go.sum.tidycheck

# Guard against local replace directives in go.mod
replace-check:
	@if grep -q '^[[:space:]]*replace[[:space:]]' go.mod; then \
		echo "ERROR: go.mod contains replace directives"; \
		grep '^[[:space:]]*replace[[:space:]]' go.mod; \
		echo ""; \
		echo "Remove replace directives before releasing."; \
		exit 1; \
	fi
	@echo "Replace check passed (no local replace directives)"

# --- Security targets ---

# Run vulnerability scanner
vuln:
	@echo "Running govulncheck..."
	govulncheck ./...

# Run secret scanner
secrets:
	@command -v gitleaks >/dev/null || (echo "Install gitleaks: brew install gitleaks" && exit 1)
	gitleaks detect --source . --verbose

# Run all security checks
security: lint vuln secrets

# Lint GitHub Actions workflows
lint-actions:
	actionlint
	zizmor .

# Full suite: everything CI runs.
check-all: fmt-check vet lint lint-actions test-race bench tidy-check

# Full pre-flight for release
release-check: check-all replace-check vuln secrets

# Cut a release (delegates to scripts/release.sh)
release:
	DRY_RUN=$(DRY_RUN) scripts/release.sh $(VERSION)

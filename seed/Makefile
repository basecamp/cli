BINARY_NAME := $(shell basename $(CURDIR))
BUILD_DIR := ./bin

# PGO (Profile-Guided Optimization)
PGO_PROFILE := default.pgo
HAS_PGO := $(shell test -f $(PGO_PROFILE) && echo 1 || echo 0)
ifeq ($(HAS_PGO),1)
    PGO_FLAGS := -pgo=$(PGO_PROFILE)
else
    PGO_FLAGS :=
endif

# Set to legacy name pattern when doing a rename (e.g. LEGACY_PATTERN ?= oldname|OLDNAME)
LEGACY_PATTERN ?=

.DEFAULT_GOAL := check

.PHONY: check check-toolchain build test test-race test-e2e vet lint fmt fmt-check bench bench-cpu bench-mem \
	check-all clean tidy tidy-check check-naming replace-check check-surface check-surface-diff \
	check-surface-compat vuln secrets security provenance-check release-check release \
	sync-skills sync-skills-remote collect-profile build-pgo test-coverage coverage

# Default target: fast checks suitable for pre-commit / inner-loop dev.
check: fmt-check vet test test-e2e tidy-check

# Guard against Go toolchain mismatch (mise environment)
check-toolchain:
	@GOV=$$(go version | awk '{print $$3}'); \
	ROOT=$$(go env GOROOT); \
	ROOTV=$$($$ROOT/bin/go version | awk '{print $$3}'); \
	if [ "$$GOV" != "$$ROOTV" ]; then \
		echo "ERROR: Go toolchain mismatch"; \
		echo "  PATH go:   $$GOV ($$(which go))"; \
		echo "  GOROOT go: $$ROOTV ($$ROOT/bin/go)"; \
		echo "Fix: eval \"\$$(mise hook-env)\" && make <target>"; \
		exit 1; \
	fi

build: check-toolchain
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

# Build with PGO optimization (requires default.pgo)
build-pgo:
	@if [ ! -f $(PGO_PROFILE) ]; then \
		echo "Warning: $(PGO_PROFILE) not found. Run 'make collect-profile' first."; \
		echo "Building without PGO..."; \
		go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME); \
	else \
		echo "Building with PGO optimization..."; \
		go build $(PGO_FLAGS) -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME); \
	fi

test: check-toolchain
	go test ./...

test-race: check-toolchain
	go test -race ./...

test-e2e:
	bats e2e/

vet:
	go vet ./...

lint:
	golangci-lint run

fmt:
	gofmt -w .

fmt-check:
	@test -z "$$(gofmt -l .)" || (echo "Run 'make fmt' to fix formatting" && gofmt -l . && exit 1)

# Run tests with coverage
test-coverage: check-toolchain
	go test -v -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html

# Coverage with browser open
coverage: test-coverage
	@command -v open >/dev/null 2>&1 && open coverage.html || true

bench: check-toolchain
	go test -bench=. -benchmem ./...

bench-cpu: check-toolchain
	go test -bench=. -cpuprofile=cpu.prof ./...

bench-mem: check-toolchain
	go test -bench=. -memprofile=mem.prof ./...

tidy:
	go mod tidy

# Verify go.mod/go.sum are tidy (CI gate).
# Restores original files on any failure so the check is non-mutating.
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

# Guard against stale legacy references. Set LEGACY_PATTERN to enable.
check-naming:
	@if [ -z "$(LEGACY_PATTERN)" ]; then \
		echo "Naming check skipped (LEGACY_PATTERN not set)"; \
		exit 0; \
	fi; \
	HITS=$$(rg -n --hidden --type-add 'bats:*.bats' -t go -t sh -t yaml -t json -t md -t bats -t toml '$(LEGACY_PATTERN)' -g '!.git/' . 2>/dev/null) || true; \
	if [ -f .naming-allowlist ] && [ -n "$$HITS" ]; then \
		while IFS= read -r line; do \
			line=$${line%%\#*}; \
			line=$$(echo "$$line" | sed 's/^[[:space:]]*//;s/[[:space:]]*$$//'); \
			[ -z "$$line" ] && continue; \
			HITS=$$(echo "$$HITS" | grep -v -F "$$line" || true); \
		done < .naming-allowlist; \
	fi; \
	if [ -n "$$HITS" ]; then \
		echo "ERROR: Legacy references matching '$(LEGACY_PATTERN)' found outside allowlist:"; \
		echo "$$HITS"; \
		echo ""; \
		echo "Either rename the reference or add the path to .naming-allowlist"; \
		exit 1; \
	fi; \
	echo "Naming check passed (no stale references)"

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

# Generate CLI surface snapshot (validates binary produces valid output)
check-surface: build
	@command -v jq >/dev/null 2>&1 || { \
		echo "ERROR: jq is required for check-surface but was not found."; \
		echo "Install with: brew install jq (macOS), apt-get install jq (Debian/Ubuntu)"; \
		exit 1; \
	}
	scripts/check-cli-surface.sh $(BUILD_DIR)/$(BINARY_NAME) /tmp/cli-surface.txt
	@echo "CLI surface snapshot generated ($$(wc -l < /tmp/cli-surface.txt) entries)"

# Compare CLI surface against baseline (fails on removals)
check-surface-diff:
	scripts/check-cli-surface-diff.sh $(BASELINE) $(CURRENT)

# Check CLI surface compatibility against previous tag (mirrors CI gate)
check-surface-compat: build
	@scripts/check-cli-surface.sh $(BUILD_DIR)/$(BINARY_NAME) /tmp/current-surface.txt
	@PREV_TAG=$$(git describe --tags --abbrev=0 HEAD^ 2>/dev/null || echo ""); \
	if [ -n "$$PREV_TAG" ]; then \
		SCRIPT_DIR="$$(pwd)/scripts"; \
		BINARY_NAME="$(BINARY_NAME)"; \
		BASELINE_DIR=$$(mktemp -d); \
		cleanup() { git worktree remove "$$BASELINE_DIR" --force 2>/dev/null || true; rm -rf "$$BASELINE_DIR" 2>/dev/null || true; }; \
		trap cleanup EXIT; \
		git worktree add "$$BASELINE_DIR" "$$PREV_TAG" || { echo "Failed to create worktree for $$PREV_TAG"; exit 1; }; \
		cd "$$BASELINE_DIR" && make build && \
		"$$SCRIPT_DIR/check-cli-surface.sh" "./bin/$$BINARY_NAME" /tmp/baseline-surface.txt; \
		cd - >/dev/null; \
		cleanup; trap - EXIT; \
		scripts/check-cli-surface-diff.sh /tmp/baseline-surface.txt /tmp/current-surface.txt; \
	else \
		echo "First release — no baseline to compare against"; \
	fi

# --- Security targets ---

vuln:
	@echo "Running govulncheck..."
	govulncheck ./...

secrets:
	@command -v gitleaks >/dev/null || (echo "Install gitleaks: brew install gitleaks" && exit 1)
	gitleaks detect --source . --verbose

security: lint vuln secrets

# Uncomment and configure for SDK provenance tracking:
# SDK_MODULE := github.com/your-org/your-sdk
# SDK_PROVENANCE := internal/version/sdk-provenance.json
provenance-check:
	@echo "Provenance check: not configured (see Makefile)"

# Full pre-flight for release: check + replace-check + vuln + secrets + race + surface compat + tidy
release-check: check replace-check vuln secrets test-race check-surface-compat tidy-check

# Cut a release (delegates to scripts/release.sh)
release:
	DRY_RUN=$(DRY_RUN) scripts/release.sh $(VERSION)

# Sync skills to distribution repo (local dry-run)
# Usage: make sync-skills TAG=v1.2.3
sync-skills:
	@test -n "$(TAG)" || (echo "Usage: make sync-skills TAG=v1.2.3" && exit 1)
	RELEASE_TAG=$(TAG) SOURCE_SHA=$$(git rev-parse HEAD) DRY_RUN=local scripts/sync-skills.sh

# Sync skills (dry-run against real target repo)
# Usage: make sync-skills-remote TAG=v1.2.3 SKILLS_TOKEN=ghp_...
sync-skills-remote:
	@test -n "$(TAG)" || (echo "Usage: make sync-skills-remote TAG=v1.2.3 SKILLS_TOKEN=..." && exit 1)
	@test -n "$(SKILLS_TOKEN)" || (echo "Usage: make sync-skills-remote TAG=v1.2.3 SKILLS_TOKEN=..." && exit 1)
	RELEASE_TAG=$(TAG) SOURCE_SHA=$$(git rev-parse HEAD) DRY_RUN=remote SKILLS_TOKEN=$(SKILLS_TOKEN) scripts/sync-skills.sh

# Collect PGO profile from benchmarks
collect-profile:
	./scripts/collect-profile.sh

# Full suite: everything CI runs. Slower — includes lint, race detector, benchmarks, surface.
check-all: fmt-check vet lint test-race test-e2e bench check-surface

clean:
	rm -rf $(BUILD_DIR)

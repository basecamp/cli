BINARY_NAME := $(shell basename $(CURDIR))
BUILD_DIR := ./bin

.PHONY: check build test test-race test-e2e vet lint fmt fmt-check bench bench-cpu bench-mem check-all clean

# Default target: fast checks suitable for pre-commit / inner-loop dev.
check: fmt-check vet test test-e2e

build:
	go build -o $(BUILD_DIR)/$(BINARY_NAME) ./cmd/$(BINARY_NAME)

test:
	go test ./...

test-race:
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

bench:
	go test -bench=. -benchmem ./...

bench-cpu:
	go test -bench=. -cpuprofile=cpu.prof ./...

bench-mem:
	go test -bench=. -memprofile=mem.prof ./...

# Full suite: everything CI runs. Slower — includes lint, race detector, benchmarks.
check-all: fmt-check vet lint test-race test-e2e bench

clean:
	rm -rf $(BUILD_DIR)

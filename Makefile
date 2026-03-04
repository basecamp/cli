.PHONY: check test test-race vet lint fmt fmt-check bench check-all

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

# Full suite: everything CI runs.
check-all: fmt-check vet lint test-race bench

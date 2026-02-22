# Console Channel — build, test, lint.
# Usable as subdirectory of opentalon/opentalon or as standalone repo (own go.mod).

.PHONY: build test lint

BINARY_NAME ?= console

build:
	go build -o $(BINARY_NAME) ./cmd/console
	@echo "Built: $(BINARY_NAME)"

test:
	go test -race -count=1 ./...

lint:
	golangci-lint run

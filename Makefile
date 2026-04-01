## Build binary
build:
	go build -o bin/deck ./cmd/deck

## Run tests
test:
	go test ./... -v

## Run linter
lint:
	golangci-lint run ./...

## Install locally
install: build
	cp bin/deck $(GOPATH)/bin/deck

## Show help
help:
	@awk -F'[ :]' '/^##/ {comment=$$0; gsub(/^##[ ]*/, "", comment)} !/^help:/ && /^([A-Za-z_-]+):/ && !seen[$$1]++ {printf "  %-20s %s\n", $$1, (comment ? "- " comment : ""); comment=""} !/^##/ {comment=""}' Makefile

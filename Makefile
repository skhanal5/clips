GOLANGCI_LINT := $(shell go env GOPATH)/bin/golangci-lint

.PHONY: setup fmt vet lint build test check clean tools

setup: tools
	git config core.hooksPath .githooks

tools:
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install golang.org/x/tools/cmd/goimports@latest

fmt:
	go fmt ./...

vet:
	go vet ./...

lint:
	$(GOLANGCI_LINT) run ./...

build:
	go build ./...

test:
	go test ./... -race -count=1

check: fmt vet lint build test

clean:
	go clean

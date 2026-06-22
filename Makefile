SHELL := bash

.PHONY: all check test fmt

all: check test

check:
	./scripts/milestone-check.sh

test:
	go test ./...

fmt:
	find . -name '*.go' -print0 | xargs -0 gofmt -w

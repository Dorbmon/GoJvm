SHELL := bash

.PHONY: all check test fmt runtime

all: check test

check:
	./scripts/milestone-check.sh

test:
	go test ./...

runtime:
	./scripts/build-gojvm-runtime.sh

fmt:
	find . -name '*.go' -print0 | xargs -0 gofmt -w

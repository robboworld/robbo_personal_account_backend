# Go runs in Docker by default (no Go toolchain needed on the host).
# Use a local toolchain instead: make test GO=go
GO_IMAGE ?= golang:1.24
GO ?= docker run --rm -v $(CURDIR):/src -w /src \
	-v robbo-gomod:/go/pkg/mod -v robbo-gocache:/root/.cache/go-build \
	-e GOFLAGS=-buildvcs=false $(GO_IMAGE) go

.PHONY: build vet test check test-external test-integration

build:
	$(GO) build ./...

vet:
	$(GO) vet ./...

# Unit tests: no network, no Docker.
test:
	$(GO) test -short -count=1 ./...

check: build vet test

# Calls the live edx test API.
test-external:
	$(GO) test -tags external -count=1 ./package/edx/...

# Needs Docker for dockertest (with the default GO, mount /var/run/docker.sock).
test-integration:
	$(GO) test -tags integration -count=1 ./tests/...

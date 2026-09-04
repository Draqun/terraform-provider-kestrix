# Makefile for the Kestrix Terraform/OpenTofu provider.

BINARY    := bin/terraform-provider-kestrix
GO        := go
VERSION   ?= dev

# Pinned development tools. staticcheck must stay compatible with the Go
# release in go.mod: newer staticcheck releases raise their minimum Go version
# and `@latest` then fails to install against the pinned toolchain.
STATICCHECK_VERSION ?= v0.7.0

# Portable Go toolchain locations. Override with env vars if your system needs
# different paths; otherwise everything stays inside the repo so this Makefile
# works on any machine without touching the user's $HOME.
GOCACHE     ?= $(CURDIR)/.gocache
GOMODCACHE  ?= $(CURDIR)/.gomodcache
GOPATH      ?= $(CURDIR)/.gopath
GOBIN       ?= $(GOPATH)/bin
GOENV       := GOCACHE=$(GOCACHE) GOMODCACHE=$(GOMODCACHE) GOPATH=$(GOPATH) GOBIN=$(GOBIN) PATH=$(GOBIN):$(PATH)

.PHONY: help build test coverage protocol docs acceptance release fmt fmt-check vet staticcheck lint quality clean

## help: Show this help message
help:
	@echo "Kestrix provider targets:"
	@grep -E '^## ' $(MAKEFILE_LIST) | sed 's/## /  /'

## build: Build the provider binary into bin/
build:
	@mkdir -p bin
	$(GOENV) $(GO) build -ldflags "-X main.version=$(VERSION)" -o $(BINARY) .

## test: Run unit tests
test:
	$(GOENV) $(GO) test ./... -count=1

## coverage: Run unit tests and enforce the 100% statement gate
coverage:
	$(GOENV) bash scripts/coverage.sh

## protocol: Start the provider and negotiate with Terraform and OpenTofu
protocol: build
	bash scripts/protocol.sh

## docs: Validate the examples and documentation
docs:
	bash scripts/docs.sh

## acceptance: Run the opt-in lifecycle tests against a running Kestrix
acceptance: build
	bash scripts/acceptance.sh

## release: Build reproducible archives and checksums for every platform
release:
	bash scripts/release.sh $(VERSION)

## fmt: Format Go sources and Terraform examples
fmt:
	$(GOENV) $(GO) fmt ./...
	@command -v terraform >/dev/null 2>&1 && terraform fmt -recursive examples || true

## fmt-check: Check formatting
fmt-check:
	@echo "Checking formatting..."
	@out=$$(gofmt -l $(shell find . -name '*.go' -not -path './.gomodcache/*' -not -path './.gocache/*' -not -path './.gopath/*') 2>/dev/null); \
	if [ -n "$$out" ]; then echo "Files need formatting:"; echo "$$out"; exit 1; fi
	@echo "✓ Formatting OK"

## vet: Run go vet
vet:
	$(GOENV) $(GO) vet ./...

## staticcheck: Run staticcheck
staticcheck:
	@set -eu; \
	if [ ! -x "$(GOBIN)/staticcheck" ]; then \
		echo "Installing staticcheck..."; \
		$(GOENV) $(GO) install honnef.co/go/tools/cmd/staticcheck@$(STATICCHECK_VERSION); \
	fi
	$(GOENV) $(GOBIN)/staticcheck ./...

## lint: Run all linters
lint: fmt-check vet staticcheck

## quality: Run linters, tests, protocol and documentation checks
quality: lint coverage protocol docs

## clean: Remove build output
clean:
	rm -rf bin dist coverage.out

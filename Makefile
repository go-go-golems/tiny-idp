.PHONY: test build lint lintmax fmt-check vuln verify logcopter-generate logcopter-check glazed-lint-build glazed-lint bump-go-go-golems

GO_PACKAGES ?= ./...
LOGCOPTER_PACKAGES ?= ./cmd/... ./internal/... ./pkg/...
LOGCOPTER_AREA_PREFIX ?= tinyidp
LOGCOPTER_STRIP_PREFIX ?= github.com/manuel/tinyidp

GOLANGCI_LINT_VERSION ?= $(shell cat .golangci-lint-version 2>/dev/null || echo v2.12.2)
GO_TOOLCHAIN_VERSION ?= $(shell GOWORK=off go env GOVERSION)
GOLANGCI_LINT_BIN ?= /tmp/golangci-lint-$(GOLANGCI_LINT_VERSION)-$(GO_TOOLCHAIN_VERSION)

GOVULNCHECK_VERSION ?= v1.5.0
GOVULNCHECK_BIN ?= /tmp/govulncheck-$(GOVULNCHECK_VERSION)

GLAZED_LINT_BIN ?= /tmp/glazed-lint-$(GLAZED_VERSION)-$(GO_TOOLCHAIN_VERSION)
GLAZED_LINT_PKG ?= github.com/go-go-golems/glazed/cmd/tools/glazed-lint
GLAZED_VERSION ?= $(shell GOWORK=off go list -m -f '{{.Version}}' github.com/go-go-golems/glazed 2>/dev/null)
GLAZED_LINT_TOOL_VERSION ?= $(if $(GLAZED_VERSION),$(GLAZED_VERSION),latest)
GLAZED_LINT_DIRS ?= ./cmd/... ./internal/... ./pkg/...
GLAZED_LINT_FLAGS ?= -glazedclilint.allow-paths=cmd/tinyidp/main.go,internal/cmds/admin.go,internal/cmds/admin_backup.go,internal/cmds/admin_client.go,internal/cmds/admin_export.go,internal/cmds/admin_keys.go,internal/cmds/admin_ops.go,internal/cmds/config.go,internal/cmds/profiles.go

test:
	GOWORK=off go test $(GO_PACKAGES)

build:
	GOWORK=off go build $(GO_PACKAGES)

$(GOVULNCHECK_BIN):
	@echo "Installing govulncheck $(GOVULNCHECK_VERSION)"
	GOBIN=/tmp GOWORK=off go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	@mv /tmp/govulncheck $(GOVULNCHECK_BIN)

vuln: $(GOVULNCHECK_BIN)
	GOWORK=off $(GOVULNCHECK_BIN) $(GO_PACKAGES)

verify: build test lint vuln

$(GOLANGCI_LINT_BIN):
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)"
	GOBIN=/tmp GOWORK=off go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@mv /tmp/golangci-lint $(GOLANGCI_LINT_BIN)

golangci-lint-install: $(GOLANGCI_LINT_BIN)

lint: glazed-lint-build golangci-lint-install
	GOWORK=off $(GOLANGCI_LINT_BIN) run -v
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)

lintmax: glazed-lint-build golangci-lint-install
	GOWORK=off $(GOLANGCI_LINT_BIN) run -v --max-same-issues=100
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)

fmt-check: golangci-lint-install
	GOWORK=off $(GOLANGCI_LINT_BIN) fmt --diff

logcopter-generate:
	GOWORK=off go generate ./...

logcopter-check:
	GOWORK=off go tool logcopter-gen -area-prefix $(LOGCOPTER_AREA_PREFIX) -strip-prefix $(LOGCOPTER_STRIP_PREFIX) -check $(LOGCOPTER_PACKAGES)

glazed-lint-build:
	@if [ -n "$(GLAZED_VERSION)" ] && [ "$(GLAZED_VERSION)" != "(devel)" ]; then \
		echo "Installing $(GLAZED_LINT_PKG)@$(GLAZED_VERSION)"; \
		GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_VERSION) || { \
			echo "Falling back to $(GLAZED_LINT_PKG)@$(GLAZED_LINT_TOOL_VERSION)"; \
			GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_LINT_TOOL_VERSION); \
		}; \
	else \
		echo "Installing $(GLAZED_LINT_PKG)@$(GLAZED_LINT_TOOL_VERSION)"; \
		GOBIN=$(dir $(GLAZED_LINT_BIN)) GOWORK=off go install $(GLAZED_LINT_PKG)@$(GLAZED_LINT_TOOL_VERSION); \
	fi
	@mv $(dir $(GLAZED_LINT_BIN))glazed-lint $(GLAZED_LINT_BIN)

glazed-lint: glazed-lint-build
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)

bump-go-go-golems:
	@deps="$$(awk '/^require[[:space:]]+github\.com\/go-go-golems\// { print $$2 } /^[[:space:]]*github\.com\/go-go-golems\// { print $$1 }' go.mod | sort -u)"; \
	if [ -z "$$deps" ]; then \
		echo "No github.com/go-go-golems dependencies in go.mod"; \
	else \
		echo "Bumping go-go-golems dependencies:"; \
		echo "$$deps"; \
		for dep in $$deps; do GOWORK=off go get "$${dep}@latest"; done; \
	fi
	GOWORK=off go mod tidy

.PHONY: test build lint lintmax fmt fmt-check gosec vuln verify auditlint logcopter-generate logcopter-check docs-export goreleaser tag-major tag-minor tag-patch release install glazed-lint-build glazed-lint idpui-analyzer-build idpui-analyzer bump-go-go-golems

GO_PACKAGES ?= ./...
LOGCOPTER_PACKAGES ?= ./cmd/... ./internal/... ./pkg/...
LOGCOPTER_AREA_PREFIX ?= tinyidp
LOGCOPTER_STRIP_PREFIX ?= github.com/go-go-golems/tiny-idp

VERSION ?= dev
GORELEASER_ARGS ?= --skip=sign --snapshot --clean
GORELEASER_TARGET ?= --single-target
DOCSCTL_OUTPUT ?= .docsctl/tinyidp-help.sqlite

GOLANGCI_LINT_VERSION ?= $(shell cat .golangci-lint-version 2>/dev/null || echo v2.12.2)
GO_TOOLCHAIN_VERSION ?= $(shell GOWORK=off go env GOVERSION)
GOLANGCI_LINT_BIN ?= /tmp/golangci-lint-$(GOLANGCI_LINT_VERSION)-$(GO_TOOLCHAIN_VERSION)

GOVULNCHECK_VERSION ?= v1.5.0
GOVULNCHECK_BIN ?= /tmp/govulncheck-$(GOVULNCHECK_VERSION)
GOSEC_VERSION ?= v2.22.10
GOSEC_BIN ?= /tmp/gosec-$(GOSEC_VERSION)-$(GO_TOOLCHAIN_VERSION)

GLAZED_LINT_BIN ?= /tmp/glazed-lint-$(GLAZED_VERSION)-$(GO_TOOLCHAIN_VERSION)
GLAZED_LINT_PKG ?= github.com/go-go-golems/glazed/cmd/tools/glazed-lint
GLAZED_VERSION ?= $(shell GOWORK=off go list -m -f '{{.Version}}' github.com/go-go-golems/glazed 2>/dev/null)
GLAZED_LINT_TOOL_VERSION ?= $(if $(GLAZED_VERSION),$(GLAZED_VERSION),latest)
GLAZED_LINT_DIRS ?= ./cmd/... ./internal/... ./pkg/...
GLAZED_LINT_FLAGS ?= -glazedclilint.allow-paths=cmd/tinyidp/main.go,internal/cmds/admin.go,internal/cmds/admin_backup.go,internal/cmds/admin_client.go,internal/cmds/admin_export.go,internal/cmds/admin_keys.go,internal/cmds/admin_ops.go,internal/cmds/config.go,internal/cmds/profiles.go
IDPUI_ANALYZER_BIN ?= /tmp/tinyidp-idpui-analyzer
IDPUI_ANALYZER_PKG ?= ./ttmp/2026/07/13/TINYIDP-UI-001--secure-customizable-login-and-consent-renderer/scripts/idpui_analyzer/cmd/idpui-analyzer
IDPUI_ANALYZER_DIRS ?= ./pkg/idpui/... ./internal/fositeadapter ./cmd/tinyidp-xapp/internal/loginui
AUDITLINT_PKG ?= ./ttmp/2026/07/09/TINYIDP-PROD-REVIEW-001--production-readiness-review-for-tiny-idp/scripts/auditlint
AUDITLINT_DIRS ?= ./pkg/... ./internal/... ./cmd/tinyidp-xapp/... ./examples/...

test:
	GOWORK=off go test $(GO_PACKAGES)

build:
	GOWORK=off go build $(GO_PACKAGES)

fmt:
	GOWORK=off go fmt $(GO_PACKAGES)

$(GOVULNCHECK_BIN):
	@echo "Installing govulncheck $(GOVULNCHECK_VERSION)"
	GOBIN=/tmp GOWORK=off go install golang.org/x/vuln/cmd/govulncheck@$(GOVULNCHECK_VERSION)
	@mv /tmp/govulncheck $(GOVULNCHECK_BIN)

vuln: $(GOVULNCHECK_BIN)
	GOWORK=off $(GOVULNCHECK_BIN) $(GO_PACKAGES)

$(GOSEC_BIN):
	@echo "Installing gosec $(GOSEC_VERSION)"
	GOBIN=/tmp GOWORK=off go install github.com/securego/gosec/v2/cmd/gosec@$(GOSEC_VERSION)
	@mv /tmp/gosec $(GOSEC_BIN)

gosec: $(GOSEC_BIN)
	GOWORK=off $(GOSEC_BIN) -quiet -exclude-generated -exclude=G101,G204,G304,G301,G306 -exclude-dir=ttmp ./...

verify: build test lint auditlint gosec vuln

auditlint:
	@for package in $(AUDITLINT_DIRS); do \
		GOWORK=off GOFLAGS=-buildvcs=false go run $(AUDITLINT_PKG) "$$package" || exit $$?; \
	done

$(GOLANGCI_LINT_BIN):
	@echo "Installing golangci-lint $(GOLANGCI_LINT_VERSION)"
	GOBIN=/tmp GOWORK=off go install github.com/golangci/golangci-lint/v2/cmd/golangci-lint@$(GOLANGCI_LINT_VERSION)
	@mv /tmp/golangci-lint $(GOLANGCI_LINT_BIN)

golangci-lint-install: $(GOLANGCI_LINT_BIN)

lint: glazed-lint-build golangci-lint-install idpui-analyzer-build
	GOWORK=off $(GOLANGCI_LINT_BIN) run -v
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)
	GOWORK=off go vet -vettool=$(IDPUI_ANALYZER_BIN) $(IDPUI_ANALYZER_DIRS)

lintmax: glazed-lint-build golangci-lint-install
	GOWORK=off $(GOLANGCI_LINT_BIN) run -v --max-same-issues=100
	GOWORK=off go vet -vettool=$(GLAZED_LINT_BIN) $(GLAZED_LINT_FLAGS) $(GLAZED_LINT_DIRS)

fmt-check: golangci-lint-install
	GOWORK=off $(GOLANGCI_LINT_BIN) fmt --diff

logcopter-generate:
	GOWORK=off go generate ./...

logcopter-check:
	GOWORK=off go tool logcopter-gen -area-prefix $(LOGCOPTER_AREA_PREFIX) -strip-prefix $(LOGCOPTER_STRIP_PREFIX) -check $(LOGCOPTER_PACKAGES)

docs-export:
	@mkdir -p $(dir $(DOCSCTL_OUTPUT))
	GOWORK=off go run ./cmd/tinyidp help export --format sqlite --output-path $(DOCSCTL_OUTPUT)

goreleaser:
	GOWORK=off goreleaser release $(GORELEASER_ARGS) $(GORELEASER_TARGET)

tag-major:
	git tag $(shell svu major)

tag-minor:
	git tag $(shell svu minor)

tag-patch:
	git tag $(shell svu patch)

release:
	git push origin --tags
	GOWORK=off GOPROXY=proxy.golang.org go list -m github.com/go-go-golems/tiny-idp@$(shell svu current)

install:
	GOWORK=off go install ./cmd/tinyidp

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

idpui-analyzer-build:
	GOWORK=off go build -o $(IDPUI_ANALYZER_BIN) $(IDPUI_ANALYZER_PKG)

idpui-analyzer: idpui-analyzer-build
	GOWORK=off go vet -vettool=$(IDPUI_ANALYZER_BIN) $(IDPUI_ANALYZER_DIRS)

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

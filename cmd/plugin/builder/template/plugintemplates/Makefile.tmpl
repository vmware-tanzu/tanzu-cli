ROOT_DIR_RELATIVE := .

include $(ROOT_DIR_RELATIVE)/common.mk
include $(ROOT_DIR_RELATIVE)/plugin-tooling.mk

TOOLS_DIR := tools
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin
GOLANGCI_LINT := $(TOOLS_BIN_DIR)/golangci-lint
GOLANGCI_LINT_VERSION := 1.52.2

.PHONY: lint
lint: $(GOLANGCI_LINT) ## Lint the plugin
	$(GOLANGCI_LINT) run -v

.PHONY: gomod
gomod: ## Update go module dependencies
	go mod tidy

.PHONY: test
test:
	go test ./...

$(TOOLS_BIN_DIR):
	-mkdir -p $@

$(GOLANGCI_LINT): $(TOOLS_BIN_DIR) ## Install golangci-lint
	curl -L https://github.com/golangci/golangci-lint/releases/download/v$(GOLANGCI_LINT_VERSION)/golangci-lint-$(GOLANGCI_LINT_VERSION)-$(GOHOSTOS)-$(GOHOSTARCH).tar.gz | tar -xz -C /tmp/
	mv /tmp/golangci-lint-$(GOLANGCI_LINT_VERSION)-$(GOHOSTOS)-$(GOHOSTARCH)/golangci-lint $(@)

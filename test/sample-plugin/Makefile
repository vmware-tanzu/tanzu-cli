# Copyright 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

ROOT_DIR_RELATIVE := .

include $(ROOT_DIR_RELATIVE)/common.mk
include $(ROOT_DIR_RELATIVE)/plugin-tooling.mk

ROOT_DIR := $(shell git rev-parse --show-toplevel)
GO := go
GINKGO := ${ROOT_DIR}/hack/tools/bin/ginkgo
GOTEST_VERBOSE ?= -v

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

.PHONY: install-builder
install-builder: ## Install builder
	$(MAKE) -C $(ROOT_DIR_RELATIVE)/../.. prepare-builder
	unset BUILDER_PLUGIN ; \
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	$(MAKE) -C $(ROOT_DIR_RELATIVE)/../.. plugin-build-install-local PLUGIN_NAME=builder

.PHONEY: e2e-tests-simple-plugin ## Run all e2e tests for simple plugin
e2e-tests-simple-plugin: install-builder plugin-build-local plugin-install-local e2e-tests-sample-plugin-functionality e2e-tests-sample-plugin-e2e-api

.PHONEY: e2e-tests-sample-plugin-functionality ## Run e2e tests for sample plugin funtionality
e2e-tests-sample-plugin-functionality:
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GINKGO} --keep-going --output-dir $(ROOT_DIR_RELATIVE)/testresults --json-report=results.json --keep-separate-reports --race --nodes=1 ${GOTEST_VERBOSE} -r $(ROOT_DIR_RELATIVE)/cmd/plugin/sample-plugin/test/e2e/plugin_functionality  --trace > /tmp/out && { cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; } || { exit_code=$$?; cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; exit $$exit_code; } \


.PHONEY: e2e-tests-sample-plugin-e2e-api ## Run e2e tests to demonstrate CLI E2E API
e2e-tests-sample-plugin-e2e-api:
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GINKGO} --keep-going --output-dir $(ROOT_DIR_RELATIVE)/testresults --json-report=results.json --keep-separate-reports --race --nodes=1 ${GOTEST_VERBOSE} -r $(ROOT_DIR_RELATIVE)/cmd/plugin/sample-plugin/test/e2e/framework_functionality  --trace > /tmp/out && { cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; } || { exit_code=$$?; cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; exit $$exit_code; } \

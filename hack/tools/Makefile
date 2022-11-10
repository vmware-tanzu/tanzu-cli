# Copyright 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# Build tooling that is used for the main project

# Ensure Make is run with bash shell as some syntax below is bash-specific
SHELL := /usr/bin/env bash

.DEFAULT_GOAL := help

# Use GOPROXY environment variable if set
GOPROXY := $(shell go env GOPROXY)
ifeq (,$(strip $(GOPROXY)))
GOPROXY := https://proxy.golang.org
endif
export GOPROXY

# Directories.
ROOT_DIR:=$(shell git rev-parse --show-toplevel)
BIN_DIR := bin

GOIMPORTS_VERSION=0.1.12
VALE_VERSION=2.20.1
GOLANGCI_LINT_VERSION=1.46.0
MISSPELL_VERSION=0.3.4

# Host information.
HOST_OS=$(shell go env GOOS)
HOST_ARCH=$(shell go env GOARCH)

# Binaries.
GOIMPORTS      	   := $(BIN_DIR)/goimports
VALE               := $(BIN_DIR)/vale
GOLANGCI_LINT      := $(BIN_DIR)/golangci-lint
MISSPELL           := $(BIN_DIR)/misspell
## --------------------------------------
## Help
## --------------------------------------

help: ## Display this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## Binaries
## --------------------------------------

goimports: $(GOIMPORTS) ## Install goimports
$(GOIMPORTS):
	mkdir -p $(BIN_DIR)
	GOBIN=$(ROOT_DIR)/hack/tools/$(BIN_DIR) go install golang.org/x/tools/cmd/goimports@v$(GOIMPORTS_VERSION)

golangci-lint: $(GOLANGCI_LINT) ## Install golangci-lint
$(GOLANGCI_LINT):
	mkdir -p $(BIN_DIR)
	GOBIN=$(ROOT_DIR)/hack/tools/$(BIN_DIR) go install github.com/golangci/golangci-lint/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION)

vale: $(VALE)
$(VALE):
	mkdir -p $(BIN_DIR)
	# vale uses 'macOS' for darwin, 'Linux' for linux, '64-bit' for amd64 (yet arm64 remains unchanged)
	# for its release asset naming, so manually map them
	curl -sfL https://github.com/errata-ai/vale/releases/download/v$(VALE_VERSION)/vale_$(VALE_VERSION)_$(shell echo $(HOST_OS) | \
		sed 's,darwin,macOS,g'| sed 's,linux,Linux,g')_$(shell echo $(HOST_ARCH) | sed 's,amd64,64-bit,g').tar.gz | tar -xz -C /tmp/
	mv /tmp/vale $(@)
	chmod a+x $(@)

misspell: $(MISSPELL) ## Install misspell
$(MISSPELL):
	mkdir -p $(BIN_DIR)
	GOBIN=$(ROOT_DIR)/hack/tools/$(BIN_DIR) go install  github.com/client9/misspell/cmd/misspell@v$(MISSPELL_VERSION)

## --------------------------------------
## Cleanup
## --------------------------------------

.PHONY: clean
clean: ## Remove all generated binaries
	rm -rf bin

# Copyright 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# Ensure Make is run with bash shell as some syntax below is bash-specific
SHELL := /usr/bin/env bash

ROOT_DIR := $(shell git rev-parse --show-toplevel)
ARTIFACTS_DIR ?= $(ROOT_DIR)/artifacts
ARTIFACTS_ADMIN_DIR ?= $(ROOT_DIR)/artifacts-admin

XDG_CONFIG_HOME := ${HOME}/.config
export XDG_CONFIG_HOME
# Local path to publish the tanzu CLI plugins
TANZU_PLUGIN_PUBLISH_PATH ?= $(XDG_CONFIG_HOME)/_tanzu-plugins

# Golang specific variables
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOHOSTOS ?= $(shell go env GOHOSTOS)
GOHOSTARCH ?= $(shell go env GOHOSTARCH)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
GO := go

GOTEST_VERBOSE ?= -v

# Directories
TOOLS_DIR := $(abspath $(ROOT_DIR)/hack/tools)
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin

# Add tooling binaries here and in hack/tools/Makefile
GOIMPORTS          := $(TOOLS_BIN_DIR)/goimports
GOLANGCI_LINT      := $(TOOLS_BIN_DIR)/golangci-lint
VALE               := $(TOOLS_BIN_DIR)/vale
MISSPELL           := $(TOOLS_BIN_DIR)/misspell
TOOLING_BINARIES   := $(GOIMPORTS) $(GOLANGCI_LINT) $(VALE) $(MISSPELL)

# Build and version information

NUL = /dev/null
ifeq ($(GOHOSTOS),windows)
	NUL = NUL
endif
BUILD_SHA ?= $$(git describe --match=$(git rev-parse --short HEAD) --always --dirty)
BUILD_DATE ?= $$(date -u +"%Y-%m-%d")
BUILD_VERSION ?= $$(git describe --tags --abbrev=0 2>$(NUL))

ifeq ($(strip $(BUILD_VERSION)),)
BUILD_VERSION = dev
endif
LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-cli/pkg/buildinfo.Date=$(BUILD_DATE)'
LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-cli/pkg/buildinfo.SHA=$(BUILD_SHA)'
LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-cli/pkg/buildinfo.Version=$(BUILD_VERSION)'

# On the same build the core CLI and plugins in this repo will be built with
# the same build info, but said info is plumbed into the plugins differently.
PLUGIN_LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-plugin-runtime/plugin/buildinfo.Date=$(BUILD_DATE)'
PLUGIN_LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-plugin-runtime/plugin/buildinfo.SHA=$(BUILD_SHA)'
PLUGIN_LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-plugin-runtime/plugin/buildinfo.Version=$(BUILD_VERSION)'

# Add supported OS-ARCHITECTURE combinations here
ENVS ?= linux-amd64 windows-amd64 darwin-amd64

CLI_TARGETS := $(addprefix build-cli-,${ENVS})
PLUGIN_TARGETS := $(addprefix build-plugin-admin-,${ENVS})
ADMIN_PLUGINS ?= builder test

## --------------------------------------
## Help
## --------------------------------------

help: ## Display this help (default)
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m\033[32m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## All
## --------------------------------------

.PHONY: all
all: gomod build-all test lint ## Run all major targets (lint, test, build)

## --------------------------------------
## Build
## --------------------------------------

.PHONY: cross-build
cross-build: ${CLI_TARGETS} ${PLUGIN_TARGETS} publish-admin-plugins-all-local## Build the Tanzu Core CLI and plugins for all supported platforms and also publish admin plugins locally

.PHONY: build-all
build-all: build-cli-${GOHOSTOS}-${GOHOSTARCH} build-publish-admin-plugins-local ## Build the Tanzu Core CLI, admin plugins and publish admin plugins locally for the local platform

.PHONY: build
build: build-cli-${GOHOSTOS}-${GOHOSTARCH} ## Build the Tanzu Core CLI for the local platform

.PHONY: build-publish-admin-plugins-local
build-publish-admin-plugins-local: build-plugin-admin-${GOHOSTOS}-${GOHOSTARCH}  publish-admin-plugins-local

build-cli-%: ##Build the Tanzu Core CLI for a platform
	$(eval ARCH = $(word 2,$(subst -, ,$*)))
	$(eval OS = $(word 1,$(subst -, ,$*)))

	@echo build $(OS)-$(ARCH) CLI with version: $(BUILD_VERSION)

	@if [ "$(filter $(OS)-$(ARCH),$(ENVS))" = "" ]; then\
		printf "\n\n======================================\n";\
		printf "! $(OS)-$(ARCH) is not an officially supported platform!\n";\
		printf "======================================\n\n";\
	fi

	@if [ "$(OS)" = "windows" ]; then \
		GOOS=$(OS) GOARCH=$(ARCH) $(GO) build --ldflags "$(LD_FLAGS)"  -o "$(ARTIFACTS_DIR)/$(OS)/$(ARCH)/cli/core/$(BUILD_VERSION)/tanzu-cli-$(OS)_$(ARCH).exe" ./cmd/tanzu/main.go;\
	else \
		GOOS=$(OS) GOARCH=$(ARCH) $(GO) build --ldflags "$(LD_FLAGS)"  -o "$(ARTIFACTS_DIR)/$(OS)/$(ARCH)/cli/core/$(BUILD_VERSION)/tanzu-cli-$(OS)_$(ARCH)" ./cmd/tanzu/main.go;\
	fi


## --------------------------------------
## Plugins-specific
## --------------------------------------

BUILDER := $(ROOT_DIR)/bin/builder
BUILDER_SRC := $(shell find cmd/plugin/builder -type f -print)
$(BUILDER): $(BUILDER_SRC)
	cd cmd/plugin/builder && $(GO) build -o $(BUILDER) .

.PHONY: prepare-builder
prepare-builder: $(BUILDER) ## Build Tanzu CLI builder plugin

build-plugin-admin-%: prepare-builder
	$(eval ARCH = $(word 2,$(subst -, ,$*)))
	$(eval OS = $(word 1,$(subst -, ,$*)))

	@if [ "$(filter $(OS)-$(ARCH),$(ENVS))" = "" ]; then\
		printf "\n\n======================================\n";\
		printf "! $(OS)-$(ARCH) is not an officially supported platform!\n";\
		printf "======================================\n\n";\
	fi

	@echo build $(OS)-$(ARCH) plugin with version: $(BUILD_VERSION)
	$(BUILDER) cli compile --version $(BUILD_VERSION) --ldflags "$(PLUGIN_LD_FLAGS)" --path ./cmd/plugin --artifacts "$(ARTIFACTS_ADMIN_DIR)/$(OS)/$(ARCH)/cli" --target ${OS}_${ARCH}


.PHONY: publish-admin-plugins-all-local
publish-admin-plugins-all-local: prepare-builder ## Publish CLI admin plugins locally for all supported os-arch
	$(BUILDER) publish --type local --plugins "$(ADMIN_PLUGINS)" --version $(BUILD_VERSION) --os-arch "${ENVS}" --local-output-discovery-dir "$(TANZU_PLUGIN_PUBLISH_PATH)/discovery/admin" --local-output-distribution-dir "$(TANZU_PLUGIN_PUBLISH_PATH)/distribution" --input-artifact-dir $(ARTIFACTS_ADMIN_DIR)

.PHONY: publish-admin-plugins-local
publish-admin-plugins-local: prepare-builder ## Publish CLI admin plugins locally for current host os-arch only
	$(BUILDER) publish --type local --plugins "$(ADMIN_PLUGINS)" --version $(BUILD_VERSION) --os-arch "${GOHOSTOS}-${GOHOSTARCH}" --local-output-discovery-dir "$(TANZU_PLUGIN_PUBLISH_PATH)/discovery/admin" --local-output-distribution-dir "$(TANZU_PLUGIN_PUBLISH_PATH)/distribution" --input-artifact-dir $(ARTIFACTS_ADMIN_DIR)

## --------------------------------------
## OS Packages
## --------------------------------------
.PHONY: apt-package
apt-package: ## Build a debian package to use with APT
	@if [ "$$(command -v docker)" == "" ]; then \
		echo "Docker required to build apt package" ;\
		exit 1 ;\
	fi

	@# To call this target, the VERSION variable must be set by the caller.  The version must match an existing release
	@# of the tanzu CLI on Github. E.g., VERSION=v0.26.0 make apt-package
	docker run --rm -e VERSION=$${VERSION} -v $(ROOT_DIR):$(ROOT_DIR) ubuntu $(ROOT_DIR)/hack/apt/build_package.sh

.PHONY: rpm-package
rpm-package: ## Build an RPM package
	@if [ "$$(command -v docker)" == "" ]; then \
		echo "Docker required to build rpm package" ;\
		exit 1 ;\
	fi

	@# To call this target, the VERSION variable must be set by the caller.  The version must match an existing release
	@# of the tanzu CLI on Github. E.g., VERSION=v0.26.0 make rpm-package
	docker run --rm -e VERSION=$${VERSION} -v $(ROOT_DIR):$(ROOT_DIR) fedora $(ROOT_DIR)/hack/rpm/build_package.sh

.PHONY: choco-package
choco-package: ## Build a Chocolatey package
	@if [ "$$(command -v docker)" = "" ]; then \
		echo "Docker required to build chocolatey package" ;\
		exit 1 ;\
	fi

	@# There are only AMD64 images to run chocolatey on docker
	@if [ "$(GOHOSTARCH)" != "amd64" ]; then \
		echo "Can only build chocolatey package on an amd64 machine at the moment" ;\
		exit 1 ;\
	fi

	@# To call this target, the VERSION variable must be set by the caller.  The version must match an existing release
	@# of the tanzu CLI on Github. E.g., VERSION=v0.26.0 make choco-package
	docker run --rm -e VERSION=$${VERSION} -v $(ROOT_DIR):$(ROOT_DIR) chocolatey/choco $(ROOT_DIR)/hack/choco/build_package.sh

## --------------------------------------
## Testing
## --------------------------------------

.PHONY: test
test: fmt ## Run Tests
	${GO} test ./... -timeout 60m -race -coverprofile coverage.txt ${GOTEST_VERBOSE}

.PHONY: fmt
fmt: $(GOIMPORTS) ## Run goimports
	$(GOIMPORTS) -w -local github.com/vmware-tanzu ./

lint: tools go-lint doc-lint misspell yamllint ## Run linting and misspell checks
	# Check licenses in shell scripts and Makefiles
	hack/check/check-license.sh

.PHONY: gomod
gomod: ## Update go module dependencies
	go mod tidy

misspell: $(MISSPELL)
	hack/check/misspell.sh

yamllint:
	hack/check/check-yaml.sh

go-lint: $(GOLANGCI_LINT)  ## Run linting of go source
	$(GOLANGCI_LINT) run --timeout=10m || exit 1

	# Prevent use of deprecated ioutils module
	@CHECK=$$(grep -r --include="*.go"  --exclude="zz_generated*" ioutil .); \
	if [ -n "$${CHECK}" ]; then \
		echo "ioutil is deprecated, use io or os replacements"; \
		echo "https://go.dev/doc/go1.16#ioutil"; \
		echo "$${CHECK}"; \
		exit 1; \
	fi

doc-lint: $(VALE) ## Run linting checks for docs
	$(VALE) --config=.vale/config.ini --glob='*.md' ./
	# mdlint rules with possible errors and fixes can be found here:
	# https://github.com/DavidAnson/markdownlint/blob/main/doc/Rules.md
	# Additional configuration can be found in the .markdownlintrc file.
	hack/check/check-mdlint.sh

.PHONY: generate-fakes
generate-fakes: ## Generate fakes for writing unit tests
	$(GO) generate ./...
	$(MAKE) fmt

.PHONY: verify
verify: gomod fmt ## Run all verification scripts
	./hack/check/check-dirty.sh

## --------------------------------------
## Tooling Binaries
## --------------------------------------

tools: $(TOOLING_BINARIES) ## Build tooling binaries
.PHONY: $(TOOLING_BINARIES)
$(TOOLING_BINARIES):
	make -C $(TOOLS_DIR) $(@F)

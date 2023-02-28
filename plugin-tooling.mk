# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

# Ensure Make is run with bash shell as some syntax below is bash-specific
SHELL := /usr/bin/env bash

# Build and version information

GOHOSTOS ?= $(shell go env GOHOSTOS)
GOHOSTARCH ?= $(shell go env GOHOSTARCH)

NUL = /dev/null
ifeq ($(GOHOSTOS),windows)
	NUL = NUL
endif
PLUGIN_BUILD_SHA ?= $(shell git describe --match=$(git rev-parse --short HEAD) --always --dirty)
PLUGIN_BUILD_DATE ?= $(shell date -u +"%Y-%m-%d")
PLUGIN_BUILD_VERSION ?= $(shell git describe --tags 2>$(NUL))

ifeq ($(strip $(PLUGIN_BUILD_VERSION)),)
PLUGIN_BUILD_VERSION = dev
endif
PLUGIN_LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-plugin-runtime/plugin/buildinfo.Date=$(PLUGIN_BUILD_DATE)'
PLUGIN_LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-plugin-runtime/plugin/buildinfo.SHA=$(PLUGIN_BUILD_SHA)'
PLUGIN_LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-plugin-runtime/plugin/buildinfo.Version=$(PLUGIN_BUILD_VERSION)'

# Add supported OS-ARCHITECTURE combinations here
PLUGIN_BUILD_OS_ARCH ?= linux-amd64 windows-amd64 darwin-amd64

# Paths and Directory information
ROOT_DIR := $(shell git rev-parse --show-toplevel)

PLUGIN_DIR := ./cmd/plugin
PLUGIN_BINARY_ARTIFACTS_DIR := $(ROOT_DIR)/artifacts/plugins
PLUGIN_PACKAGE_ARTIFACTS_DIR := $(ROOT_DIR)/artifacts/packages

REGISTRY_PORT := 5001
REGISTRY_ENDPOINT := localhost:$(REGISTRY_PORT)
PLUGIN_NAME := "*"

# Repository specific configuration
BUILDER := $(ROOT_DIR)/bin/builder
PUBLISHER := tkg
VENDOR := vmware
PLUGIN_PUBLISH_REPOSITORY := localhost:$(REGISTRY_PORT)/test/v1/tanzu-cli/plugins

## --------------------------------------
## Plugin Build and Publish Tooling
## --------------------------------------

PLUGIN_BUILD_TARGETS := $(addprefix plugin-build-,${PLUGIN_BUILD_OS_ARCH})

.PHONY: plugin-build
plugin-build: $(PLUGIN_BUILD_TARGETS) ## Build all plugin binaries for all supported os-arch

plugin-build-local: plugin-build-$(GOHOSTOS)-$(GOHOSTARCH) ## Build all plugin binaries for local platform
	
plugin-build-%:
	$(eval ARCH = $(word 2,$(subst -, ,$*)))
	$(eval OS = $(word 1,$(subst -, ,$*)))
	$(BUILDER) plugin build \
		--path $(PLUGIN_DIR) \
		--binary-artifacts $(PLUGIN_BINARY_ARTIFACTS_DIR) \
		--version $(PLUGIN_BUILD_VERSION) \
		--ldflags "$(PLUGIN_LD_FLAGS)" \
		--os-arch $(OS)_$(ARCH) \
		--match $(PLUGIN_NAME)

.PHONY: plugin-build-packages
plugin-build-packages: local-registry ## Build plugin packages
	$(BUILDER) plugin build-package \
		--binary-artifacts $(PLUGIN_BINARY_ARTIFACTS_DIR) \
		--package-artifacts $(PLUGIN_PACKAGE_ARTIFACTS_DIR) \
		--oci-registry $(REGISTRY_ENDPOINT)

.PHONY: plugin-publish-packages
plugin-publish-packages: ## Publish plugin packages
	$(BUILDER) plugin publish-package \
		--package-artifacts $(PLUGIN_PACKAGE_ARTIFACTS_DIR) \
		--publisher $(PUBLISHER) \
		--vendor $(VENDOR) \
		--repository $(PLUGIN_PUBLISH_REPOSITORY)

.PHONY: plugin-build-and-publish-packages
plugin-build-and-publish-packages: plugin-build plugin-build-packages plugin-publish-packages ## Build and Publish plugin packages

## --------------------------------------
## docker
## --------------------------------------

.PHONY: local-registry
local-registry: clean-registry ## Starts up a local docker registry for generating packages
	docker run -d -p $(REGISTRY_PORT):5000 --name temp-package-registry mirror.gcr.io/library/registry:2

.PHONY: clean-registry
clean-registry: ## Stops and removes local docker registry
	docker stop temp-package-registry && docker rm -v temp-package-registry || true

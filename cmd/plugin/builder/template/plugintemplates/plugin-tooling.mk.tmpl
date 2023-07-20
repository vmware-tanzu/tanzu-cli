# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

GOHOSTOS ?= $(shell go env GOHOSTOS)
GOHOSTARCH ?= $(shell go env GOHOSTARCH)

NUL = /dev/null
ifeq ($(GOHOSTOS),windows)
	NUL = NUL
endif

# Build and version information
PLUGIN_BUILD_SHA ?= $(shell git describe --match=$(git rev-parse --short HEAD) --always --dirty)
PLUGIN_BUILD_DATE ?= $(shell date -u +"%Y-%m-%d")
PLUGIN_BUILD_VERSION ?= $(shell git describe --tags --abbrev=0 2>$(NUL))

ifeq ($(strip $(PLUGIN_BUILD_VERSION)),)
PLUGIN_BUILD_VERSION = v0.0.0
endif
PLUGIN_LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-plugin-runtime/plugin/buildinfo.Date=$(PLUGIN_BUILD_DATE)'
PLUGIN_LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-plugin-runtime/plugin/buildinfo.SHA=$(PLUGIN_BUILD_SHA)'
PLUGIN_LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-plugin-runtime/plugin/buildinfo.Version=$(PLUGIN_BUILD_VERSION)'
PLUGIN_GO_FLAGS ?=

# Add supported OS-ARCHITECTURE combinations here
PLUGIN_BUILD_OS_ARCH ?= linux-amd64 windows-amd64 darwin-amd64

# Paths and Directory information
ROOT_DIR := $(shell git rev-parse --show-toplevel)

PLUGIN_DIR := ./cmd/plugin
PLUGIN_BINARY_ARTIFACTS_DIR := $(ROOT_DIR)/artifacts/plugins
PLUGIN_PACKAGE_ARTIFACTS_DIR := $(ROOT_DIR)/artifacts/packages
PLUGIN_MANIFEST_FILE := $(PLUGIN_PACKAGE_ARTIFACTS_DIR)/plugin_manifest.yaml
PLUGIN_GROUP_MANIFEST_FILE := $(PLUGIN_BINARY_ARTIFACTS_DIR)/plugin_group_manifest.yaml

REGISTRY_PORT ?= 5001
REGISTRY_ENDPOINT ?= localhost:$(REGISTRY_PORT)
PLUGIN_NAME ?= *

# Repository specific configuration
TZBIN ?= tanzu
BUILDER_PLUGIN ?= $(TZBIN) builder
PUBLISHER ?= tzcli
VENDOR ?= vmware
PLUGIN_PUBLISH_REPOSITORY ?= $(REGISTRY_ENDPOINT)/test/v1/tanzu-cli/plugins
PLUGIN_INVENTORY_IMAGE_TAG ?= latest

PLUGIN_SCOPE_ASSOCIATION_FILE ?= ""
PLUGIN_GROUP_NAME_VERSION ?= # e.g. default:v1.0.0, app-developer:v0.1.0
# Get the name and version of the plugin group from the PLUGIN_GROUP_NAME_VERSION variable
# if the PLUGIN_GROUP_NAME/PLUGIN_GROUP_VERSION are not already set.  This is to make it
# easier on the plugin developer to allow having to set one less variable.
TMP=$(subst :, ,${PLUGIN_GROUP_NAME_VERSION})
PLUGIN_GROUP_NAME ?= $(word 1,${TMP})
PLUGIN_GROUP_VERSION ?= $(word 2,${TMP})

# A plugin group description is only needed if a new group is being defined.
# If we are publishing a new version of an existing group, the description is optional.
PLUGIN_GROUP_DESCRIPTION ?=
PLUGIN_GROUP_DESCRIPTION_FLAG_AND_VALUE =
ifneq ($(strip $(PLUGIN_GROUP_DESCRIPTION)),)
PLUGIN_GROUP_DESCRIPTION_FLAG_AND_VALUE = --description "$(PLUGIN_GROUP_DESCRIPTION)"
endif

# Process configuration and setup additional variables
TANZU_BUILDER_OVERRIDE ?=
OVERRIDE_FLAG = 
ifneq ($(strip $(TANZU_BUILDER_OVERRIDE)),)
OVERRIDE_FLAG = --override
endif

## --------------------------------------
## Plugin Build and Publish Tooling
## --------------------------------------

PLUGIN_BUILD_TARGETS := $(addprefix plugin-build-,${PLUGIN_BUILD_OS_ARCH})

.PHONY: plugin-build-install-local ## Build and Install all plugins using local plugin artifacts directory
plugin-build-install-local: plugin-build-local plugin-install-local

.PHONY: plugin-install-local ## Install all plugins from local plugin artifacts directory
plugin-install-local:
	tanzu plugin install all --local $(PLUGIN_BINARY_ARTIFACTS_DIR)/$(GOHOSTOS)/$(GOHOSTARCH)

.PHONY: plugin-build
plugin-build: $(PLUGIN_BUILD_TARGETS) generate-plugin-bundle ## Build all plugin binaries for all supported os-arch

plugin-build-local: plugin-build-$(GOHOSTOS)-$(GOHOSTARCH) ## Build all plugin binaries for local platform
	
plugin-build-%:
	$(eval ARCH = $(word 2,$(subst -, ,$*)))
	$(eval OS = $(word 1,$(subst -, ,$*)))
	$(BUILDER_PLUGIN) plugin build \
		--path $(PLUGIN_DIR) \
		--binary-artifacts $(PLUGIN_BINARY_ARTIFACTS_DIR) \
		--version $(PLUGIN_BUILD_VERSION) \
		--ldflags "$(PLUGIN_LD_FLAGS)" \
		--goflags "$(PLUGIN_GO_FLAGS)" \
		--os-arch $(OS)_$(ARCH) \
		--match "$(PLUGIN_NAME)" \
		--plugin-scope-association-file $(PLUGIN_SCOPE_ASSOCIATION_FILE)

.PHONY: plugin-build-packages
plugin-build-packages: ## Build plugin packages
	$(BUILDER_PLUGIN) plugin build-package \
		--binary-artifacts $(PLUGIN_BINARY_ARTIFACTS_DIR) \
		--package-artifacts $(PLUGIN_PACKAGE_ARTIFACTS_DIR)

.PHONY: plugin-publish-packages
plugin-publish-packages: plugin-build-packages plugin-local-registry ## Publish plugin packages
	$(BUILDER_PLUGIN) plugin publish-package \
		--package-artifacts $(PLUGIN_PACKAGE_ARTIFACTS_DIR) \
		--publisher $(PUBLISHER) \
		--vendor $(VENDOR) \
		--repository $(PLUGIN_PUBLISH_REPOSITORY)

.PHONY: plugin-build-and-publish-packages
plugin-build-and-publish-packages: plugin-build plugin-publish-packages ## Build and Publish plugin packages

.PHONY: inventory-init
inventory-init: ## Initialize empty plugin inventory
	$(BUILDER_PLUGIN) inventory init \
		--repository $(PLUGIN_PUBLISH_REPOSITORY) \
		--plugin-inventory-image-tag $(PLUGIN_INVENTORY_IMAGE_TAG) \
		$(OVERRIDE_FLAG)

.PHONY: inventory-plugin-add
inventory-plugin-add: ## Add plugins to the inventory database
	$(BUILDER_PLUGIN) inventory plugin add \
		--repository $(PLUGIN_PUBLISH_REPOSITORY) \
		--plugin-inventory-image-tag $(PLUGIN_INVENTORY_IMAGE_TAG) \
		--publisher $(PUBLISHER) \
		--vendor $(VENDOR) \
		--manifest $(PLUGIN_MANIFEST_FILE)

.PHONY: inventory-plugin-activate
inventory-plugin-activate: ## Activate plugins in the inventory database
	$(BUILDER_PLUGIN) inventory plugin activate \
		--repository $(PLUGIN_PUBLISH_REPOSITORY) \
		--plugin-inventory-image-tag $(PLUGIN_INVENTORY_IMAGE_TAG) \
		--publisher $(PUBLISHER) \
		--vendor $(VENDOR) \
		--manifest $(PLUGIN_MANIFEST_FILE)

.PHONY: inventory-plugin-deactivate
inventory-plugin-deactivate: ## Deactivate plugins in the inventory database
	$(BUILDER_PLUGIN) inventory plugin deactivate \
		--repository $(PLUGIN_PUBLISH_REPOSITORY) \
		--plugin-inventory-image-tag $(PLUGIN_INVENTORY_IMAGE_TAG) \
		--publisher $(PUBLISHER) \
		--vendor $(VENDOR) \
		--manifest $(PLUGIN_MANIFEST_FILE)

.PHONY: inventory-plugin-group-add
inventory-plugin-group-add: ## Add plugin-group to the inventory database. Requires PLUGIN_GROUP_NAME_VERSION
	$(BUILDER_PLUGIN) inventory plugin-group add \
		--repository $(PLUGIN_PUBLISH_REPOSITORY) \
		--plugin-inventory-image-tag $(PLUGIN_INVENTORY_IMAGE_TAG) \
		--publisher $(PUBLISHER) \
		--vendor $(VENDOR) \
		--manifest $(PLUGIN_GROUP_MANIFEST_FILE) \
		--name $(PLUGIN_GROUP_NAME) \
		--version $(PLUGIN_GROUP_VERSION) \
		$(PLUGIN_GROUP_DESCRIPTION_FLAG_AND_VALUE) \
		$(OVERRIDE_FLAG)

.PHONY: inventory-plugin-group-activate
inventory-plugin-group-activate: ## Activate plugin-group in the inventory database. Requires PLUGIN_GROUP_NAME_VERSION
	$(BUILDER_PLUGIN) inventory plugin-group activate \
		--repository $(PLUGIN_PUBLISH_REPOSITORY) \
		--plugin-inventory-image-tag $(PLUGIN_INVENTORY_IMAGE_TAG) \
		--publisher $(PUBLISHER) \
		--vendor $(VENDOR) \
		--name $(PLUGIN_GROUP_NAME) \
		--version $(PLUGIN_GROUP_VERSION)

.PHONY: inventory-plugin-group-deactivate
inventory-plugin-group-deactivate: ## Deactivate plugin-group in the inventory database. Requires PLUGIN_GROUP_NAME_VERSION
	$(BUILDER_PLUGIN) inventory plugin-group deactivate \
		--repository $(PLUGIN_PUBLISH_REPOSITORY) \
		--plugin-inventory-image-tag $(PLUGIN_INVENTORY_IMAGE_TAG) \
		--publisher $(PUBLISHER) \
		--vendor $(VENDOR) \
		--name $(PLUGIN_GROUP_NAME) \
		--version $(PLUGIN_GROUP_VERSION)

## --------------------------------------
## docker
## --------------------------------------

.PHONY: plugin-local-registry
plugin-local-registry: plugin-clean-registry ## Starts up a local docker registry for generating packages
	docker run -d -p $(REGISTRY_PORT):5000 --name temp-package-registry mirror.gcr.io/library/registry:2.7.1

.PHONY: plugin-clean-registry
plugin-clean-registry: ## Stops and removes local docker registry
	docker stop temp-package-registry && docker rm -v temp-package-registry || true

## --------------------------------------
## Helpers
## --------------------------------------

generate-plugin-bundle:
	cd $(PLUGIN_BINARY_ARTIFACTS_DIR) && tar -czvf ../plugin_bundle.tar.gz .

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
BIN_DIR := $(shell echo "bin")

# Host information.
HOST_OS=$(shell go env GOOS)
HOST_ARCH=$(shell go env GOARCH)

# Binaries.

CONTROLLER_TOOLS_VERSION=0.9.2
COSIGN_VERSION=2.0.3-0.20230519173114-f21081a18209
GINKGO_VERSION=v2.15.0
GOIMPORTS_VERSION=0.13.0
GOLANGCI_LINT_VERSION=1.54.2
IMGPKG_VERSION=v0.35.0
KBLD_VERSION=v0.38.1
KCTRL_VERSION=v0.48.0
KIND_VERSION=v0.17.0
KUBECTL_VERSION=v1.26.0
MISSPELL_VERSION=0.3.4
VALE_VERSION=2.20.1
VENDIR_VERSION=v0.39.0
YTT_VERSION=v0.46.2
YQ_VERSION=v4.40.2

CONTROLLER_GEN     := $(BIN_DIR)/controller-gen
COSIGN             := $(BIN_DIR)/cosign
GINKGO             := $(BIN_DIR)/ginkgo
GOIMPORTS      	   := $(BIN_DIR)/goimports
GOJUNITREPORT	   := $(BIN_DIR)/go-junit-report
GOLANGCI_LINT      := $(BIN_DIR)/golangci-lint
IMGPKG             := $(BIN_DIR)/imgpkg
KBLD               := $(BIN_DIR)/kbld
KCTRL              := $(BIN_DIR)/kctrl
KIND               := $(BIN_DIR)/kind
KUBECTL            := $(BIN_DIR)/kubectl
MISSPELL           := $(BIN_DIR)/misspell
VALE               := $(BIN_DIR)/vale
VENDIR             := $(BIN_DIR)/vendir
YTT                := $(BIN_DIR)/ytt
YQ                 := $(BIN_DIR)/yq

VAL_URL := https://github.com/errata-ai/vale/releases/download/v$(VALE_VERSION)/vale_$(VALE_VERSION)_$(shell echo $(HOST_OS) | \
		sed 's,darwin,macOS,g'| sed 's,linux,Linux,g'| sed 's,windows,Windows,g')_$(shell echo $(HOST_ARCH) | sed 's,amd64,64-bit,g')$(shell echo $(HOST_OS) | \
		sed 's,darwin,.tar.gz,g'| sed 's,linux,.tar.gz,g'| sed 's,windows,.zip,g')


ifeq ($(GITHUB_ACTIONS),true)
    # Use the values from GitHub Actions
    GOPATH := $(shell echo $$GOPATH)
    GOBIN := $(shell echo $$GOBIN)
	GITHUB_INFO := Running in GitHub Actions
else
    # Use the system's GOPATH and GOBIN; if they're not set, use defaults
    GOPATH ?= $(HOME)/go
    GOBIN ?= $(GOPATH)/bin
	GITHUB_INFO := Not running in GitHub Actions
endif

$(info GOPATH before update: $(GOPATH))
$(info GOBIN before update: $(GOBIN))
$(info $(GITHUB_INFO))

# Normalize $(ROOT_DIR) and ensure no double slashes before appending
GOBIN := $(patsubst %/,%,$(subst \,/,${ROOT_DIR}))/hack/tools/$(BIN_DIR)

# Ensure $(GOPATH) uses forward slashes
GOPATH := $(subst \,/,$(GOPATH))


# It's best to use these echo commands in an actual recipe, not directly in the conditional logic
# For demonstration purposes, they are kept here but consider moving them into a rule's command
$(info GOBIN after update: $(GOBIN))
$(info GOPATH after update: $(GOPATH))


golangci-lint: $(GOLANGCI_LINT) ## Install golangci-lint
$(GOLANGCI_LINT):
	@echo "GOPATH current: $(GOPATH)"
	@echo "GOBIN current: $(GOBIN)"
	mkdir -p $(BIN_DIR)
	@ls $(GOPATH)
	@ls $(GOPATH)/bin
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@v$(GOLANGCI_LINT_VERSION)
	@ls $(GOPATH)
	@ls $(GOPATH)/bin
	@echo "GOBIN files: $(GOBIN)"
	@ls $(GOBIN)
#mv $(GOPATH)/$(GOLANGCI_LINT) $(GOBIN) && chmod a+x $(GOBIN)/$(notdir $(GOLANGCI_LINT))


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
	@echo "gobin in goimports: $(GOBIN)"
	mkdir -p $(BIN_DIR)
	go install golang.org/x/tools/cmd/goimports@v$(GOIMPORTS_VERSION)
	@echo "GOBIN files: $(GOBIN)"
	@ls $(GOBIN)
	
#mkdir -p $(BIN_DIR)
#GOBIN=$(ROOT_DIR)/hack/tools/$(BIN_DIR) go install golang.org/x/tools/cmd/goimports@v$(GOIMPORTS_VERSION)

#mkdir -p $(BIN_DIR)
#GO111MODULE=on go get -u -v golang.org/x/tools/cmd/goimports@v$(GOIMPORTS_VERSION)
#mv $(GOPATH)/bin/goimports $(GOIMPORTS)
#GOBIN=$(ROOT_DIR)/hack/tools/$(BIN_DIR) go install golang.org/x/tools/cmd/goimports@v$(GOIMPORTS_VERSION)


vale: $(VALE)
$(VALE):
	mkdir -p $(BIN_DIR)
	# vale uses 'macOS' for darwin, 'Linux' for linux, '64-bit' for amd64 (yet arm64 remains unchanged)
	# for its release asset naming, so manually map them
	@echo "Download URL: $(VAL_URL)"
	curl -sLf $(VAL_URL) -o /tmp/vale.zip && \
	if [ "$(HOST_OS)" = "windows" ]; then \
		unzip -o -d /tmp/ /tmp/vale.zip; \
		mv /tmp/vale.exe $(@); \
		chmod a+x $(@); \
	else \
		tar -xzf /tmp/vale.zip -C /tmp/; \
		mv /tmp/vale $(@); \
		chmod a+x $(@); \
	fi

controller-gen: $(CONTROLLER_GEN) ## Build controller-gen
$(CONTROLLER_GEN):
	mkdir -p $(BIN_DIR)
	go install sigs.k8s.io/controller-tools/cmd/controller-gen@v$(CONTROLLER_TOOLS_VERSION)

cosign: $(COSIGN) ## Install cosign
$(COSIGN):
	mkdir -p $(BIN_DIR)
	go install github.com/sigstore/cosign/v2/cmd/cosign@v$(COSIGN_VERSION)

misspell: $(MISSPELL) ## Install misspell
$(MISSPELL):
	mkdir -p $(BIN_DIR)
	go install  github.com/client9/misspell/cmd/misspell@v$(MISSPELL_VERSION)

imgpkg: $(IMGPKG) ## Install imgpkg 
$(IMGPKG):
	mkdir -p $(BIN_DIR)
	curl -LO https://github.com/vmware-tanzu/carvel-imgpkg/releases/download/$(IMGPKG_VERSION)/imgpkg-$(HOST_OS)-$(HOST_ARCH)
	mv imgpkg-$(HOST_OS)-$(HOST_ARCH) $(@)
	chmod a+x $(@)

kctrl: $(KCTRL) ## Install kctrl
$(KCTRL):
	mkdir -p $(BIN_DIR)
	curl -LO https://github.com/carvel-dev/kapp-controller/releases/download/$(KCTRL_VERSION)/kctrl-$(HOST_OS)-$(HOST_ARCH)
	mv kctrl-$(HOST_OS)-$(HOST_ARCH) $(@)
	chmod a+x $(@)

ytt: $(YTT) ## Install ytt
$(YTT):
	mkdir -p $(BIN_DIR)
	curl -LO https://github.com/carvel-dev/ytt/releases/download/$(YTT_VERSION)/ytt-$(HOST_OS)-$(HOST_ARCH)
	mv ytt-$(HOST_OS)-$(HOST_ARCH) $(@)
	chmod a+x $(@)

kbld: $(KBLD) ## Install kbld
$(KBLD):
	mkdir -p $(BIN_DIR)
	curl -LO https://github.com/carvel-dev/kbld/releases/download/$(KBLD_VERSION)/kbld-$(HOST_OS)-$(HOST_ARCH)
	mv kbld-$(HOST_OS)-$(HOST_ARCH) $(@)
	chmod a+x $(@)

vendir: $(VENDIR) ## Install kbld
$(VENDIR):
	mkdir -p $(BIN_DIR)
	curl -LO https://github.com/carvel-dev/vendir/releases/download/$(VENDIR_VERSION)/vendir-$(HOST_OS)-$(HOST_ARCH)
	mv vendir-$(HOST_OS)-$(HOST_ARCH) $(@)
	chmod a+x $(@)

yq: $(YQ) ## Install yq
$(YQ):
	mkdir -p $(BIN_DIR)
	curl -LO https://github.com/mikefarah/yq/releases/download/$(YQ_VERSION)/yq_$(HOST_OS)_$(HOST_ARCH)
	mv yq_$(HOST_OS)_$(HOST_ARCH) $(@)
	chmod a+x $(@)

kubectl: $(KUBECTL)
$(KUBECTL):
	mkdir -p $(BIN_DIR)
	curl -LO https://dl.k8s.io/release/$(KUBECTL_VERSION)/bin/$(HOST_OS)/$(HOST_ARCH)/kubectl
	mv kubectl $(@)
	chmod a+x $(@)

kind: $(KIND)
$(KIND):
	mkdir -p $(BIN_DIR)
	echo $(KIND_VERSION)
	echo $(HOST_OS)
	echo $(HOST_ARCH)
	curl -Lo ./kind https://kind.sigs.k8s.io/dl/$(KIND_VERSION)/kind-$(HOST_OS)-$(HOST_ARCH)
	mv ./kind $(@)
	chmod a+x $(@)

ginkgo: $(GINKGO) ## Install ginkgo
$(GINKGO):
	mkdir -p $(BIN_DIR)
	go install github.com/onsi/ginkgo/v2/ginkgo@$(GINKGO_VERSION)

go-junit-report: $(GOJUNITREPORT) ## Install go-junit-report
$(GOJUNITREPORT):
	mkdir -p $(BIN_DIR)
	go install github.com/jstemmer/go-junit-report@latest
	@ls $(GOBIN)

## --------------------------------------
## Cleanup
## --------------------------------------

.PHONY: clean
clean: ## Remove all generated binaries
	rm -rf bin

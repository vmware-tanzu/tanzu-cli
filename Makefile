# Copyright 2022 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

include ./plugin-tooling.mk
include ./test/e2e/Makefile

# Ensure Make is run with bash shell as some syntax below is bash-specific
SHELL := /usr/bin/env bash

ROOT_DIR := $(shell git rev-parse --show-toplevel)
ARTIFACTS_DIR ?= $(ROOT_DIR)/artifacts

XDG_CONFIG_HOME := ${HOME}/.config
export XDG_CONFIG_HOME

# Golang specific variables
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
GOHOSTOS ?= $(shell go env GOHOSTOS)
GOHOSTARCH ?= $(shell go env GOHOSTARCH)

GO_MODULES=$(shell find . -path "*/go.mod" | xargs -I _ dirname _)

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif
GO := go

GOTEST_VERBOSE ?= -v

# Allow turning off go's vcs build
BUILDVCS ?= true

# Directories
TOOLS_DIR := $(abspath $(ROOT_DIR)/hack/tools)
TOOLS_BIN_DIR := $(TOOLS_DIR)/bin

# Add tooling binaries here and in hack/tools/Makefile
GOIMPORTS          := $(TOOLS_BIN_DIR)/goimports
GOLANGCI_LINT      := $(TOOLS_BIN_DIR)/golangci-lint
VALE               := $(TOOLS_BIN_DIR)/vale
MISSPELL           := $(TOOLS_BIN_DIR)/misspell
CONTROLLER_GEN     := $(TOOLS_BIN_DIR)/controller-gen
IMGPKG             := $(TOOLS_BIN_DIR)/imgpkg
KCTRL              := $(TOOLS_BIN_DIR)/kctrl
KBLD               := $(TOOLS_BIN_DIR)/kbld
YTT                := $(TOOLS_BIN_DIR)/ytt
KUBECTL            := $(TOOLS_BIN_DIR)/kubectl
KIND               := $(TOOLS_BIN_DIR)/kind
GINKGO             := $(TOOLS_BIN_DIR)/ginkgo
COSIGN             := $(TOOLS_BIN_DIR)/cosign
GOJUNITREPORT      := $(TOOLS_BIN_DIR)/go-junit-report
VENDIR             := $(TOOLS_BIN_DIR)/vendir
YQ                 := $(TOOLS_BIN_DIR)/yq

TOOLING_BINARIES   := $(GOIMPORTS) $(GOLANGCI_LINT) $(VALE) $(MISSPELL) $(CONTROLLER_GEN) $(IMGPKG) $(KUBECTL) $(KIND) $(GINKGO) $(COSIGN) $(GOJUNITREPORT) $(KCTRL) $(YTT) $(KBLD) $(VENDIR) $(YQ)

# Build and version information

NUL = /dev/null
ifeq ($(GOHOSTOS),windows)
	NUL = NUL
endif
BUILD_SHA ?= $(shell git describe --match=$(git rev-parse --short HEAD) --exclude "test/e2e/framework/*" --always --dirty)
BUILD_DATE ?= $(shell date -u +"%Y-%m-%d")
BUILD_VERSION ?= $(shell git describe --tags --exclude "test/e2e/framework/*" --abbrev=0 2>$(NUL))

ifeq ($(strip $(BUILD_VERSION)),)
BUILD_VERSION = dev
endif
LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-cli/pkg/buildinfo.Date=$(BUILD_DATE)'
LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-cli/pkg/buildinfo.SHA=$(BUILD_SHA)'
LD_FLAGS += -X 'github.com/vmware-tanzu/tanzu-cli/pkg/buildinfo.Version=$(BUILD_VERSION)'

# Remove debug symbols to reduce binary size
# To build with debug symbols: TANZU_CLI_ENABLE_DEBUG=1
ifneq ($(strip $(TANZU_CLI_ENABLE_DEBUG)),1)
LD_FLAGS += -w -s
endif

APT_IMAGE=ubuntu
ifdef APT_BUILDER_IMAGE
APT_IMAGE=$(APT_BUILDER_IMAGE)
endif

RPM_IMAGE=fedora
ifdef RPM_BUILDER_IMAGE
RPM_IMAGE=$(RPM_BUILDER_IMAGE)
endif

CHOCO_IMAGE=chocolatey/choco:v1.4.0
ifdef CHOCO_BUILDER_IMAGE
CHOCO_IMAGE=$(CHOCO_BUILDER_IMAGE)
endif

# Add supported OS-ARCHITECTURE combinations here
ENVS ?= linux-amd64 windows-amd64 darwin-amd64 linux-arm64 windows-arm64 darwin-arm64

CLI_TARGETS := $(addprefix build-cli-,${ENVS})

## --------------------------------------
## Help
## --------------------------------------

help: ## Display this help (default)
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[0-9a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-28s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m\033[32m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

## --------------------------------------
## All
## --------------------------------------

.PHONY: all
all: gomod cross-build test lint ## Run all major targets (lint, test, cross-build)

## --------------------------------------
## Build
## --------------------------------------

.PHONY: cross-build
cross-build: ${CLI_TARGETS} prepare-builder plugin-build ## Build the Tanzu Core CLI and plugins for all supported platforms

.PHONY: cross-build-cli-and-publish-plugins
cross-build-cli-and-publish-plugins: ${CLI_TARGETS} cross-build-publish-plugins ## Build the Tanzu Core CLI and plugins for all supported platforms

.PHONY: cross-build-publish-plugins
cross-build-publish-plugins: prepare-builder plugin-build-and-publish-packages inventory-init inventory-plugin-add ## Build and publish the plugins for all supported platforms

.PHONY: build-all
build-all: build prepare-builder plugin-build-local ## Build the Tanzu Core CLI, admin plugins for the local platform

.PHONY: build
build: build-cli-${GOHOSTOS}-${GOHOSTARCH} ## Build the Tanzu Core CLI for the local platform
	mkdir -p bin
	cp $(ARTIFACTS_DIR)/$(GOHOSTOS)/$(GOHOSTARCH)/cli/core/$(BUILD_VERSION)/tanzu-cli-$(GOHOSTOS)_$(GOHOSTARCH) ./bin/tanzu

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
		GOOS=$(OS) GOARCH=$(ARCH) $(GO) build -buildvcs=$(BUILDVCS) -gcflags=all="-l" --ldflags "$(LD_FLAGS)" -o "$(ARTIFACTS_DIR)/$(OS)/$(ARCH)/cli/core/$(BUILD_VERSION)/tanzu-cli-$(OS)_$(ARCH).exe" ./cmd/tanzu/main.go;\
	else \
		GOOS=$(OS) GOARCH=$(ARCH) $(GO) build -buildvcs=$(BUILDVCS) -gcflags=all="-l" --ldflags "$(LD_FLAGS)" -o "$(ARTIFACTS_DIR)/$(OS)/$(ARCH)/cli/core/$(BUILD_VERSION)/tanzu-cli-$(OS)_$(ARCH)" ./cmd/tanzu/main.go;\
	fi

## --------------------------------------
## Plugins-specific
## --------------------------------------

.PHONY: prepare-builder
prepare-builder: ## Build Tanzu CLI builder plugin
	cd cmd/plugin/builder && $(GO) build -buildvcs=$(BUILDVCS) -o $(ROOT_DIR)/bin/builder .

## --------------------------------------
## OS Packages
## --------------------------------------
.PHONY: apt-package-only
apt-package-only: ## Build a debian package
	@if [ "$$(command -v docker)" == "" ]; then \
		echo "Docker required to build apt package" ;\
		exit 1 ;\
	fi
	docker run --rm -e VERSION=$(BUILD_VERSION) -e DEB_SIGNER=$(DEB_SIGNER) -v $(ROOT_DIR):$(ROOT_DIR) $(APT_IMAGE) $(ROOT_DIR)/hack/apt/build_package.sh

.PHONY: apt-package-repo
apt-package-repo: ## Build a debian package repo
	@if [ "$$(command -v docker)" == "" ]; then \
		echo "Docker required to build apt package" ;\
		exit 1 ;\
	fi
	docker run --rm -e VERSION=$(BUILD_VERSION) -e DEB_SIGNER=$(DEB_SIGNER) -e DEB_METADATA_BASE_URI=$(DEB_METADATA_BASE_URI) -v $(ROOT_DIR):$(ROOT_DIR) $(APT_IMAGE) $(ROOT_DIR)/hack/apt/build_package_repo.sh

.PHONY: apt-package-in-docker
apt-package-in-docker: ## Build a debian package from within a container already
	VERSION=$(BUILD_VERSION) $(ROOT_DIR)/hack/apt/build_package.sh
	VERSION=$(BUILD_VERSION) $(ROOT_DIR)/hack/apt/build_package_repo.sh

.PHONY: apt-package
apt-package: apt-package-only apt-package-repo  ## Build a debian package to use with APT

.PHONY: rpm-package-only
rpm-package-only: ## Build an RPM package
	@if [ "$$(command -v docker)" == "" ]; then \
		echo "Docker required to build rpm package" ;\
		exit 1 ;\
	fi
	docker run --rm -e VERSION=$(BUILD_VERSION) -e RPM_PACKAGE_NAME=$(RPM_PACKAGE_NAME) -e RPM_SIGNER=$(RPM_SIGNER) -v $(ROOT_DIR):$(ROOT_DIR) $(RPM_IMAGE) $(ROOT_DIR)/hack/rpm/build_package.sh

.PHONY: rpm-package-repo
rpm-package-repo: ## Build an RPM package repository
	@if [ "$$(command -v docker)" == "" ]; then \
		echo "Docker required to build rpm package" ;\
		exit 1 ;\
	fi
	docker run --rm -e VERSION=$(BUILD_VERSION) -e RPM_SIGNER=$(RPM_SIGNER) -e RPM_METADATA_BASE_URI=$(RPM_METADATA_BASE_URI) -v $(ROOT_DIR):$(ROOT_DIR) $(RPM_IMAGE) $(ROOT_DIR)/hack/rpm/build_package_repo.sh

.PHONY: rpm-package-in-docker
rpm-package-in-docker: ## Build an RPM package from within a container already
	VERSION=$(BUILD_VERSION) $(ROOT_DIR)/hack/rpm/build_package.sh

.PHONY: rpm-package-repo-in-docker
rpm-package-repo-in-docker: ## Build an RPM package repository from within a container already
	VERSION=$(BUILD_VERSION) $(ROOT_DIR)/hack/rpm/build_package_repo.sh

.PHONY: rpm-package
rpm-package: rpm-package-only rpm-package-repo  ## Build an RPM package and repository to use with YUM/DNF

.PHONY: choco-package
choco-package: ## Build a Chocolatey package
	@if [ "$$(command -v docker)" = "" ]; then \
		echo "Docker required to build chocolatey package" ;\
		exit 1 ;\
	fi

	@# There are only AMD64 images to run chocolatey on docker
	@# and even if we request an amd64 image, chocolatey will crash on arm64.
	@# This make target can ONLY be run on an AMD64 machine
	@if [ "$$(uname -m)" != "x86_64" ]; then \
		echo "Can only build chocolatey package on an amd64 machine at the moment" ;\
		exit 1 ;\
	fi
	@# The nuspec file uses a variable but variables don't seem to work anymore
	@# with chocolatey 2.0.0 so we continue using version 1.4.0
	docker run --rm \
		-e VERSION=$(BUILD_VERSION) \
		-e SHA_FOR_CHOCO_AMD64=$(SHA_FOR_CHOCO_AMD64) \
		-e SHA_FOR_CHOCO_ARM64=$(SHA_FOR_CHOCO_ARM64) \
		-v $(ROOT_DIR):$(ROOT_DIR) \
		$(CHOCO_IMAGE) $(ROOT_DIR)/hack/choco/build_package.sh

## --------------------------------------
## Carvel Package
## --------------------------------------
.PHONY: crd-package-for-test
crd-package-for-test: tools ## Build and release crd carvel package for testing
	$(ROOT_DIR)/hack/scripts/build-crd-package.sh


## --------------------------------------
## Testing
## --------------------------------------

.PHONY: test
test: fmt ## Run Tests
	${GO} test `go list ./... | grep -v test/e2e | grep -v test/coexistence` -timeout 60m -race -coverprofile coverage.txt ${GOTEST_VERBOSE}

.PHONY: test-with-summary-report ## Execute all unit test cases, process the output and present the summary report
test-with-summary-report: tools
	make test | tee ./make_test.output
	cat ./make_test.output | go tool test2json > ./test_suite_output.json
	./hack/scripts/generate-cli-ginkgo-tests-summary.sh ./test_suite_output.json > ./CLI-ginkgo-tests-summary.txt
	cat ./make_test.output | ./hack/tools/bin/go-junit-report  > ./CLI-junit-report.xml
	./hack/scripts/generate-cli-unit-tests-report.sh ./CLI-junit-report.xml ./CLI-ginkgo-tests-summary.txt
	rm ./make_test.output ./test_suite_output.json ./CLI-ginkgo-tests-summary.txt ./CLI-junit-report.xml

.PHONY: e2e-cli-core ## Execute all CLI Core E2E Tests
e2e-cli-core: tools crd-package-for-test start-test-central-repo start-airgapped-local-registry e2e-cli-core-all ## Execute all CLI Core E2E Tests

.PHONY: setup-custom-cert-for-test-central-repo
setup-custom-cert-for-test-central-repo: ## Setup up the custom ca cert for test-central-repo in the config file
	@if [ ! -d $(ROOT_DIR)/hack/central-repo/certs ]; then \
    	wget https://storage.googleapis.com/tanzu-cli/data/testcerts/local-central-repo-testcontent.bz2 -O $(ROOT_DIR)/hack/central-repo/local-central-repo-testcontent.bz2;\
  		tar xjf $(ROOT_DIR)/hack/central-repo/local-central-repo-testcontent.bz2 -C $(ROOT_DIR)/hack/central-repo/;\
	fi
	echo "Adding docker test central repo cert to the config file"
	TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" TANZU_CLI_EULA_PROMPT_ANSWER="Yes" $(ROOT_DIR)/bin/tanzu config cert delete localhost:9876 || true
	$(ROOT_DIR)/bin/tanzu config cert add --host localhost:9876 --ca-cert $(ROOT_DIR)/hack/central-repo/certs/localhost.crt

.PHONY: start-test-central-repo
start-test-central-repo: stop-test-central-repo setup-custom-cert-for-test-central-repo ## Starts up a test central repository locally with docker
	@if [ ! -d $(ROOT_DIR)/hack/central-repo/registry-content ]; then \
		(cd $(ROOT_DIR)/hack/central-repo && tar xjf registry-content.bz2 || true;) \
	fi
	@docker run --rm -d -p 9876:443 --name central \
		-v $(ROOT_DIR)/hack/central-repo/certs:/certs \
		-e REGISTRY_HTTP_ADDR=0.0.0.0:443  \
		-e REGISTRY_HTTP_TLS_CERTIFICATE=/certs/localhost.crt  \
		-e REGISTRY_HTTP_TLS_KEY=/certs/localhost.key  \
		-v $(ROOT_DIR)/hack/central-repo/registry-content:/var/lib/registry \
		$(REGISTRY_IMAGE) > /dev/null && \
	echo "Started docker test central repo with images:" && \
	$(ROOT_DIR)/hack/central-repo/upload-plugins.sh info

.PHONY: stop-test-central-repo
stop-test-central-repo: ## Stops and removes the local test central repository
	@docker container stop central > /dev/null 2>&1 && echo "Stopped docker test central repo" || true

.PHONY: start-airgapped-local-registry
start-airgapped-local-registry: stop-airgapped-local-registry
	@docker run --rm -d -p 6001:5000 --name temp-airgapped-local-registry \
		$(REGISTRY_IMAGE) > /dev/null && \
		echo "Started docker test airgapped repo at 'localhost:6001'."

	@mkdir -p $(ROOT_DIR)/hack/central-repo/auth && docker run --entrypoint htpasswd httpd:2 -Bbn ${TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH_USERNAME} ${TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH_PASSWORD} > $(ROOT_DIR)/hack/central-repo/auth/htpasswd
	@docker run --rm -d -p 6002:5000 --name temp-airgapped-local-registry-with-auth \
		-v $(ROOT_DIR)/hack/central-repo/auth:/auth \
		-e "REGISTRY_AUTH=htpasswd" \
		-e "REGISTRY_AUTH_HTPASSWD_REALM=Registry Realm" \
		-e REGISTRY_AUTH_HTPASSWD_PATH=/auth/htpasswd \
		$(REGISTRY_IMAGE) > /dev/null && \
	echo "Started docker test airgapped repo with authentication at 'localhost:6002'."

	@docker logout localhost:6002 || true

.PHONY: stop-airgapped-local-registry
stop-airgapped-local-registry:
	@docker stop temp-airgapped-local-registry temp-airgapped-local-registry-with-auth > /dev/null 2>&1 && \
	  echo "Stopping docker test airgapped repo if running..." || true


.PHONY: setup-custom-cert-for-test-cli-service
setup-custom-cert-for-test-cli-service: ## Setup up the custom ca cert for the test cli service in the config file
	@if [ ! -d $(ROOT_DIR)/hack/central-repo/certs ]; then \
		wget https://storage.googleapis.com/tanzu-cli/data/testcerts/local-central-repo-testcontent.bz2 -O $(ROOT_DIR)/hack/central-repo/local-central-repo-testcontent.bz2;\
		tar xjf $(ROOT_DIR)/hack/central-repo/local-central-repo-testcontent.bz2 -C $(ROOT_DIR)/hack/central-repo/;\
	fi
	@echo "Adding docker test cli service cert to the config file"
	@TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" TANZU_CLI_EULA_PROMPT_ANSWER="Yes" $(ROOT_DIR)/bin/tanzu config cert delete localhost:9443 &> /dev/null || true
	$(ROOT_DIR)/bin/tanzu config cert add --host localhost:9443 --ca-cert $(ROOT_DIR)/hack/central-repo/certs/localhost.crt

.PHONY: start-test-cli-service
start-test-cli-service: stop-test-cli-service setup-custom-cert-for-test-cli-service ## Starts a test CLI service locally with docker
	@docker run -d --rm --name cli-service -p 9443:443 \
		-v $(ROOT_DIR)/hack/central-repo/certs:/certs \
		-v $(ROOT_DIR)/hack/service/install.sh:/var/www/html/cli/v1/install/install.txt \
		-v $(ROOT_DIR)/hack/service/discovery:/var/www/html/cli/v1/plugin/discovery \
		-v $(ROOT_DIR)/hack/service/binaries:/var/www/html/cli/v1/binary \
		-v $(ROOT_DIR)/hack/service/cli-service.conf:/etc/nginx/conf.d/cli-service.conf \
		nginx:alpine > /dev/null && \
	echo "Started docker test cli service at 'localhost:9443'"

.PHONY: stop-test-cli-service
stop-test-cli-service: ## Stops and removes the local test CLI service
	@docker stop cli-service > /dev/null 2>&1 && echo "Stopped docker test cli service" || true

.PHONY: fmt
fmt: $(GOIMPORTS) ## Run goimports
	$(GOIMPORTS) -w -local github.com/vmware-tanzu ./

lint: tools go-lint doc-lint misspell yamllint ## Run linting and misspell checks
	# Check licenses in shell scripts and Makefiles
	hack/check/check-license.sh

.PHONY: gomod
gomod: ## Update go module dependencies
	for i in $(GO_MODULES); do \
		echo "-- Tidying $$i --"; \
		pushd $${i}; \
		$(GO) mod tidy || exit 1; \
		popd; \
	done

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
verify: gomod fmt generate ## Run all verification scripts
	./hack/check/check-dirty.sh

## --------------------------------------
## Generators
## --------------------------------------

CONTROLLER_GEN_SRC ?= "./..."

generate-controller-code: $(CONTROLLER_GEN)  ## Generate code via controller-gen
	$(CONTROLLER_GEN) $(GENERATOR) object:headerFile="$(ROOT_DIR)/hack/boilerplate.go.txt",year=$(shell date +%Y) paths="$(CONTROLLER_GEN_SRC)" $(OPTIONS)
	$(MAKE) fmt

generate-manifests:  ## Generate API manifests e.g. CRD
	$(MAKE) generate-controller-code GENERATOR=crd OPTIONS="output:crd:artifacts:config=$(ROOT_DIR)/apis/config/crd/bases" CONTROLLER_GEN_SRC=$(CONTROLLER_GEN_SRC)

generate-docs: build
	HOME=${TMPDIR}/tanzu-doc-gen TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER=no TANZU_CLI_EULA_PROMPT_ANSWER=yes TANZU_CLI_ESSENTIALS_PLUGIN_GROUP_VERSION=unused ./bin/tanzu generate-all-docs; rm -rf ${TMPDIR}/tanzu-doc-gen

generate: generate-controller-code generate-manifests generate-docs	## Generate controller code, manifests (e.g. CRD etc.) and CLI docs

## --------------------------------------
## Tooling Binaries
## --------------------------------------

tools: $(TOOLING_BINARIES) ## Build tooling binaries
.PHONY: $(TOOLING_BINARIES)
$(TOOLING_BINARIES):
	make -C $(TOOLS_DIR) $(@F)

.PHONY: clean-tools
clean-tools:
	make -C $(TOOLS_DIR) clean

## --------------------------------------
## CLI Coexistence Testing
## --------------------------------------

# CLI Coexistence related settings

ifndef TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_DIR
TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_DIR = /app/legacy-tanzu-cli
endif

ifndef TANZU_CLI_COEXISTENCE_NEW_TANZU_CLI_DIR
TANZU_CLI_COEXISTENCE_NEW_TANZU_CLI_DIR = /app/tanzu-cli
endif

ifndef TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL
TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL = localhost:9876/tanzu-cli/plugins/central:small
endif

ifndef TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_VERSION
TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_VERSION = v0.28.1
endif

ifndef TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_URL
TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_URL = https://storage.googleapis.com/tanzu-cli-installer-packages/bin/v0.28.1/tanzu-cli-linux-amd64.tar.gz # tanzu-cli-linux-amd64 v0.28.1 bundle url
endif

ifndef TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER
TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER = no
endif

ifndef TANZU_CLI_EULA_PROMPT_ANSWER
TANZU_CLI_EULA_PROMPT_ANSWER = yes
endif

.PHONY: build-cli-coexistence ## Build CLI Coexistence docker image
build-cli-coexistence: start-test-central-repo
	docker build \
		--build-arg TANZU_CLI_BUILD_VERSION=$(BUILD_VERSION) \
		--build-arg TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_DIR=$(TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_DIR) \
		--build-arg TANZU_CLI_COEXISTENCE_NEW_TANZU_CLI_DIR=$(TANZU_CLI_COEXISTENCE_NEW_TANZU_CLI_DIR) \
		--build-arg TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_VERSION=$(TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_VERSION) \
		--build-arg TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_URL=$(TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_URL) \
		--build-arg TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) \
		--build-arg TANZU_CLI_PRE_RELEASE_REPO_IMAGE=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) \
		--build-arg TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER=$(TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER) \
		--build-arg TANZU_CLI_EULA_PROMPT_ANSWER=$(TANZU_CLI_EULA_PROMPT_ANSWER) \
		-f test/e2e/coexistence/Dockerfile \
		-t cli-coexistence \
		.

.PHONY: cli-coexistence-tests ## Run CLI Coexistence tests
cli-coexistence-tests:start-test-central-repo
	docker run --rm \
	  --network host \
	  -e TANZU_CLI_BUILD_VERSION=$(BUILD_VERSION) \
	  -e TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_DIR=$(TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_DIR) \
	  -e TANZU_CLI_COEXISTENCE_NEW_TANZU_CLI_DIR=$(TANZU_CLI_COEXISTENCE_NEW_TANZU_CLI_DIR) \
	  -e TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_VERSION=$(TANZU_CLI_COEXISTENCE_LEGACY_TANZU_CLI_VERSION) \
	  -e TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) \
	  -e TANZU_CLI_PRE_RELEASE_REPO_IMAGE=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) \
	  -e TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) \
	  -e TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER=$(TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER) \
	  -e TANZU_CLI_EULA_PROMPT_ANSWER=$(TANZU_CLI_EULA_PROMPT_ANSWER) \
	  -e TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST)  \
	  -v $(ROOT_DIR):/tmp/tanzu-cli/ \
	  -v $(ROOT_DIR)/hack/central-repo/certs:/localhost_certs/ \
	  -v $(ROOT_DIR)/hack/central-repo/cosign-key-pair:/cosign-key-pair/ \
	  -w /tmp/tanzu-cli/ \
	  cli-coexistence \
	  ${GO} test ${GOTEST_VERBOSE} ./test/e2e/coexistence... --ginkgo.v --ginkgo.randomize-all --ginkgo.trace --ginkgo.json-report=testresults/coexistence_results.json

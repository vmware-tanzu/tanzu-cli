# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

E2E_TEST_OUTPUT ?= e2e-test-output.txt
E2E_TEST_TIMEOUT ?= 60m
GOTEST_VERBOSE ?= -v
ROOT_DIR := $(shell git rev-parse --show-toplevel)

ifndef TANZU_API_TOKEN
TANZU_API_TOKEN = ""
endif

ifndef TANZU_CLI_TMC_UNSTABLE_URL
TANZU_CLI_TMC_UNSTABLE_URL = ""
endif

ifndef TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL
TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL = gcr.io/eminent-nation-87317/tanzu-cli/test/v1/plugins/plugin-inventory:latest
endif

ifndef TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL
TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL = localhost:9876/tanzu-cli/plugins/central:small
endif


.PHONY: e2e-cli-core-all ## Execute all CLI Core E2E Tests
e2e-cli-core-all: e2e-cli-lifecycle e2e-cli-config e2e-cli-plugin-compatibility-test e2e-context-k8s-tests e2e-context-tmc-test e2e-cli-plugin-lifecycle-sync-test


.PHONY: e2e-cli-lifecycle ## Execute CLI life cycle specific e2e tests
e2e-cli-lifecycle:
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="Yes" ; \
	${GO} test ${ROOT_DIR}/test/e2e/cli_lifecycle -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE} ; \

.PHONY: e2e-cli-config ## Execute CLI config life cycle specific e2e tests
e2e-cli-config:
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="Yes" ; \
	${GO} test ${ROOT_DIR}/test/e2e/config -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE} ; \

.PHONY: e2e-cli-plugin-compatibility-test ## Execute CLI Core Plugin Compatibility E2E test cases
e2e-cli-plugin-compatibility-test:
	@if [ "${TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL}" = "" ]; then \
		echo "***Skipping Plugin Compatibility test cases because environment variables TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL is not set***" ; \
	else \
		export TANZU_CLI_PRE_RELEASE_REPO_IMAGE=$(TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL) ; \
		export TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST=$(TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL) ; \
		export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="Yes" ; \
		${GO} test ${ROOT_DIR}/test/e2e/plugins_compatibility -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE} ; \
	fi 

.PHONY: e2e-cli-plugin-lifecycle-sync-test ## Execute CLI Core Plugin life cycle E2E test cases
e2e-cli-plugin-lifecycle-sync-test:
	@if [ "${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL}" = "" ]; then \
		echo "***Skipping Plugin life cycle test cases because environment variables TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL is not set***" ; \
	else \
		export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
		export TANZU_CLI_PRE_RELEASE_REPO_IMAGE=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
		export TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
		export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="Yes" ; \
		${GO} test ${ROOT_DIR}/test/e2e/plugin_lifecycle -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE} ; \
		${GO} test ${ROOT_DIR}/test/e2e/plugin_sync -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE} ; \
	fi

.PHONY: e2e-context-k8s-tests ## Execute CLI context life cycle e2e tests for k8s target
e2e-context-k8s-tests:
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="Yes" ; \
	${GO} test `go list ${ROOT_DIR}/test/e2e/context/... | grep -v test/e2e/context/tmc` -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE}

## To run TMC tests, we need to set environment variables TANZU_API_TOKEN and TANZU_CLI_TMC_UNSTABLE_URL, in case of github workflow, these are set as github environment variables
.PHONY: e2e-context-tmc-test ## Execute CLI context life cycle e2e tests for tmc target
e2e-context-tmc-test:
	@if [ "${TANZU_API_TOKEN}" = "" ] || [ "$(TANZU_CLI_TMC_UNSTABLE_URL)" = "" ]; then \
		echo "***Skipping TMC specific e2e tests cases because environment variables TANZU_API_TOKEN and TANZU_CLI_TMC_UNSTABLE_URL are not set***" ; \
	else \
	    export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="Yes" ; \
		${GO} test ${ROOT_DIR}/test/e2e/context/tmc -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE} ; \
	fi

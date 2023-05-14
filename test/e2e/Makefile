# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

E2E_TEST_OUTPUT ?= ${ROOT_DIR}/e2e-test-output.txt
E2E_TEST_TIMEOUT ?= 60m
GOTEST_VERBOSE ?= -v
ROOT_DIR := $(shell git rev-parse --show-toplevel)
GO := go

ifndef TANZU_API_TOKEN
TANZU_API_TOKEN = ""
endif

ifndef TANZU_CLI_TMC_UNSTABLE_URL
TANZU_CLI_TMC_UNSTABLE_URL = ""
endif

ifndef TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL
TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL = gcr.io/eminent-nation-87317/tanzu-cli/test/v1/plugins/plugin-inventory:latest
endif

ifndef TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST
TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST = localhost:9876
endif

ifndef TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH
TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH =  ${ROOT_DIR}/hack/central-repo/certs/localhost.crt
endif

ifndef TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL
TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL = ${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST}/tanzu-cli/plugins/central:small
endif

ifndef TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH
TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH = ${ROOT_DIR}/hack/central-repo/cosign-key-pair/cosign.pub
endif

ifndef TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_IMAGE_FOR_AIRGAPPED
TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_IMAGE_FOR_AIRGAPPED = localhost:9876/tanzu-cli/plugins/airgapped:large
endif

ifndef TANZU_CLI_E2E_AIRGAPPED_REPO
TANZU_CLI_E2E_AIRGAPPED_REPO = localhost:6001/tanzu-cli/plugins/
endif

# Set the plugin group name for the plugins used to execute E2E test cases.
E2E_TEST_USE_PLGINS_FROM_PLUGIN_GROUP_FOR_TMC ?= vmware-tmc/tmc-user:v9.9.9
E2E_TEST_USE_PLGINS_FROM_PLUGIN_GROUP_FOR_K8S ?= vmware-tkg/default:v9.9.9

.PHONY: e2e-cli-core-all ## Execute all CLI Core E2E Tests
e2e-cli-core-all: e2e-cli-lifecycle e2e-cli-config e2e-plugin-compatibility-tests e2e-plugin-lifecycle-tests  e2e-plugin-sync-tmc e2e-plugin-sync-k8s e2e-context-tmc-tests e2e-context-k8s-tests e2e-airgapped-tests

.PHONY: e2e-cli-lifecycle ## Execute CLI life cycle specific e2e tests
e2e-cli-lifecycle:
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GO} test ${ROOT_DIR}/test/e2e/cli_lifecycle -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE} ; \

.PHONY: e2e-cli-config ## Execute CLI config life cycle specific e2e tests
e2e-cli-config:
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GO} test ${ROOT_DIR}/test/e2e/config -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE} ; \

.PHONY: e2e-plugin-compatibility-tests ## Execute CLI Core Plugin Compatibility E2E test cases
e2e-plugin-compatibility-tests:
	export TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL) ; \
	export TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST=$(TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL) ; \
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GO} test ${ROOT_DIR}/test/e2e/plugins_compatibility -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE} ; \

.PHONY: e2e-plugin-lifecycle-tests ## Execute CLI Core Plugin life cycle E2E test cases
e2e-plugin-lifecycle-tests:
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST} ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH} ; \
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GO} test ${ROOT_DIR}/test/e2e/plugin_lifecycle -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE} ; \

.PHONY: e2e-plugin-sync-k8s ## Execute CLI Core Plugin sync E2E test cases for k8s target
e2e-plugin-sync-k8s:
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST} ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH} ; \
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GO} test ${ROOT_DIR}/test/e2e/plugin_sync/k8s -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE} ; \


## To run TMC tests, we need to set environment variable TANZU_API_TOKEN, in case of github workflow, its been set as environment variable
.PHONY: e2e-plugin-sync-tmc ## Execute CLI Core Plugin sync E2E test cases for tmc target
e2e-plugin-sync-tmc:
	@if [ "${TANZU_API_TOKEN}" = "" ]; then \
		echo "***Skipping TMC specific plugin sync e2e test cases because environment variables TANZU_API_TOKEN is not set***" ; \
	else \
		export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
		export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH) ; \
		export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST} ; \
		export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH} ; \
		export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
		export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
		export TANZU_CLI_E2E_TEST_ENVIRONMENT="true" ; \
		export TANZU_API_TOKEN=$(TANZU_API_TOKEN) ; \
		${GO} test -p 1 ${ROOT_DIR}/test/e2e/plugin_sync/tmc -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE} ; \
	fi


## To run TMC tests, we need to set environment variables TANZU_API_TOKEN and TANZU_CLI_TMC_UNSTABLE_URL, in case of github workflow, these are set as github environment variables
.PHONY: e2e-context-tmc-tests ## Execute CLI context life cycle e2e tests for tmc target
e2e-context-tmc-tests:
	@if [ "${TANZU_API_TOKEN}" = "" ] || [ "$(TANZU_CLI_TMC_UNSTABLE_URL)" = "" ]; then \
		echo "***Skipping TMC specific e2e tests cases because environment variables TANZU_API_TOKEN and TANZU_CLI_TMC_UNSTABLE_URL are not set***" ; \
	else \
	    export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
		export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
		${GO} test ${ROOT_DIR}/test/e2e/context/tmc -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE} ; \
	fi

.PHONY: e2e-context-k8s-tests ## Execute CLI context life cycle e2e tests for k8s target
e2e-context-k8s-tests:
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST} ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH} ; \
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GO} test `go list ${ROOT_DIR}/test/e2e/context/... | grep -v test/e2e/context/tmc` -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE}

.PHONY: e2e-airgapped-tests ## Execute CLI airgapped tests
e2e-airgapped-tests:
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_IMAGE_FOR_AIRGAPPED) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST} ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH} ; \
	export TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_IMAGE_FOR_AIRGAPPED) ; \
	export TANZU_CLI_E2E_AIRGAPPED_REPO=$(TANZU_CLI_E2E_AIRGAPPED_REPO) ; \
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="Yes" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GO} test ${ROOT_DIR}/test/e2e/airgapped -timeout ${E2E_TEST_TIMEOUT} -race -coverprofile ${E2E_TEST_OUTPUT} ${GOTEST_VERBOSE} ;

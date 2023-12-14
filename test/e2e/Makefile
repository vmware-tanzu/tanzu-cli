# Copyright 2023 VMware, Inc. All Rights Reserved.
# SPDX-License-Identifier: Apache-2.0

E2E_TEST_OUTPUT ?= ${ROOT_DIR}/e2e-test-output.txt
E2E_TEST_TIMEOUT ?= 60m
GOTEST_VERBOSE ?= -v
ROOT_DIR := $(shell git rev-parse --show-toplevel)
GO := go
GINKGO := ${ROOT_DIR}/hack/tools/bin/ginkgo

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

ifndef TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH
TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH = localhost:6002/tanzu-cli/plugins/
endif

ifndef TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH_USERNAME
TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH_USERNAME = testuser
endif

ifndef TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH_PASSWORD
TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH_PASSWORD = testpassword
endif

ifndef TANZU_CLI_E2E_INPUT_CONFIG_DATA_FILE_PATH
TANZU_CLI_E2E_INPUT_CONFIG_DATA_FILE_PATH = ${ROOT_DIR}/test/e2e/config_data/input_data.yaml
endif

# Set the plugin group name for the plugins used to execute E2E test cases.
E2E_TEST_USE_PLGINS_FROM_PLUGIN_GROUP_FOR_TMC ?= vmware-tmc/tmc-user:v9.9.9
E2E_TEST_USE_PLGINS_FROM_PLUGIN_GROUP_FOR_K8S ?= vmware-tkg/default:v9.9.9

.PHONY: e2e-cli-core-all ## Execute all CLI Core E2E Tests
xe2e-cli-core-all: e2e-cli-lifecycle e2e-cli-config e2e-plugin-compatibility-tests e2e-plugin-lifecycle-tests  e2e-plugin-sync-tmc e2e-plugin-sync-k8s e2e-context-tmc-tests e2e-context-k8s-tests e2e-airgapped-tests e2e-catalog-tests e2e-extra-column-tests

e2e-cli-core-all: e2e-plugin-sync-k8s


.PHONY: e2e-cli-lifecycle ## Execute CLI life cycle specific e2e tests
e2e-cli-lifecycle:
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GINKGO} --keep-going --output-dir ${ROOT_DIR}/test/e2e/testresults --json-report=results.json --keep-separate-reports --race --nodes=1 ${GOTEST_VERBOSE} -r ${ROOT_DIR}/test/e2e/cli_lifecycle  --trace > /tmp/out && { cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; } || { exit_code=$$?; cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; exit $$exit_code; } \

.PHONY: e2e-cli-config ## Execute CLI config life cycle specific e2e tests
e2e-cli-config:
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GINKGO} --keep-going --output-dir ${ROOT_DIR}/test/e2e/testresults --json-report=results.json --keep-separate-reports --race --nodes=1 ${GOTEST_VERBOSE} -r ${ROOT_DIR}/test/e2e/config  --trace > /tmp/out && { cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; } || { exit_code=$$?; cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; exit $$exit_code; } \

.PHONY: e2e-plugin-compatibility-tests ## Execute CLI Core Plugin Compatibility E2E test cases
e2e-plugin-compatibility-tests:
	export TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL) ; \
	export TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST=$(TANZU_CLI_E2E_TEST_CENTRAL_REPO_URL) ; \
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GINKGO} --keep-going --output-dir ${ROOT_DIR}/test/e2e/testresults --json-report=results.json --keep-separate-reports --race --nodes=1 ${GOTEST_VERBOSE} -r ${ROOT_DIR}/test/e2e/plugins_compatibility  --trace > /tmp/out && { cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; } || { exit_code=$$?; cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; exit $$exit_code; } \

.PHONY: e2e-plugin-lifecycle-tests ## Execute CLI Core Plugin life cycle E2E test cases
e2e-plugin-lifecycle-tests:
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
	export TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST} ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH} ; \
	export TANZU_CLI_E2E_INPUT_CONFIG_DATA_FILE_PATH=$(TANZU_CLI_E2E_INPUT_CONFIG_DATA_FILE_PATH) ; \
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GINKGO} --keep-going --output-dir ${ROOT_DIR}/test/e2e/testresults --json-report=results.json --keep-separate-reports --race --nodes=1 ${GOTEST_VERBOSE} -r ${ROOT_DIR}/test/e2e/plugin_lifecycle  --trace > /tmp/out && { cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; } || { exit_code=$$?; cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; exit $$exit_code; } \
	

.PHONY: e2e-plugin-sync-k8s ## Execute CLI Core Plugin sync E2E test cases for k8s target
e2e-plugin-sync-k8s:
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
	export TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST} ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH} ; \
	export TANZU_CLI_E2E_INPUT_CONFIG_DATA_FILE_PATH=$(TANZU_CLI_E2E_INPUT_CONFIG_DATA_FILE_PATH) ; \
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GINKGO} -failFast --output-dir ${ROOT_DIR}/test/e2e/testresults --json-report=results.json --keep-separate-reports --race --nodes=1 ${GOTEST_VERBOSE} -r ${ROOT_DIR}/test/e2e/plugin_sync/k8s  --trace | tee /tmp/out && { cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; } || { exit_code=$$?; cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; echo rm /tmp/out; exit $$exit_code; }
	#${GINKGO} --keep-going --output-dir ${ROOT_DIR}/test/e2e/testresults --json-report=results.json --keep-separate-reports --race --nodes=1 ${GOTEST_VERBOSE} -r ${ROOT_DIR}/test/e2e/plugin_sync/k8s  --trace > /tmp/out && { cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; } || { exit_code=$$?; cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; echo rm /tmp/out; exit $$exit_code; } \


## To run TMC tests, we need to set environment variable TANZU_API_TOKEN, in case of github workflow, its been set as environment variable
.PHONY: e2e-plugin-sync-tmc ## Execute CLI Core Plugin sync E2E test cases for tmc target
e2e-plugin-sync-tmc:
	@if [ "${TANZU_API_TOKEN}" = "" ]; then \
		echo "***Skipping TMC specific plugin sync e2e test cases because environment variables TANZU_API_TOKEN is not set***" ; \
	else \
		export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
		export TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
		export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH) ; \
		export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST} ; \
		export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH} ; \
		export TANZU_CLI_E2E_INPUT_CONFIG_DATA_FILE_PATH=$(TANZU_CLI_E2E_INPUT_CONFIG_DATA_FILE_PATH) ; \
		export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
		export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
		export TANZU_CLI_E2E_TEST_ENVIRONMENT="true" ; \
		export TANZU_API_TOKEN=$(TANZU_API_TOKEN) ; \
		${GINKGO} --keep-going --output-dir ${ROOT_DIR}/test/e2e/testresults --json-report=results.json --keep-separate-reports --race --nodes=1 ${GOTEST_VERBOSE} -r ${ROOT_DIR}/test/e2e/plugin_sync/tmc  --trace > /tmp/out && { cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; } || { exit_code=$$?; cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; exit $$exit_code; } \
	fi


## To run TMC tests, we need to set environment variables TANZU_API_TOKEN and TANZU_CLI_TMC_UNSTABLE_URL, in case of github workflow, these are set as github environment variables
.PHONY: e2e-context-tmc-tests ## Execute CLI context life cycle e2e tests for tmc target
e2e-context-tmc-tests:
	@if [ "${TANZU_API_TOKEN}" = "" ] || [ "$(TANZU_CLI_TMC_UNSTABLE_URL)" = "" ]; then \
		echo "***Skipping TMC specific e2e tests cases because environment variables TANZU_API_TOKEN and TANZU_CLI_TMC_UNSTABLE_URL are not set***" ; \
	else \
		export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
		export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
		${GINKGO} --keep-going --output-dir ${ROOT_DIR}/test/e2e/testresults --json-report=results.json --keep-separate-reports --race --nodes=1 ${GOTEST_VERBOSE} -r ${ROOT_DIR}/test/e2e/context/tmc  --trace > /tmp/out && { cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; } || { exit_code=$$?; cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; exit $$exit_code; } \
	fi

.PHONY: e2e-context-k8s-tests ## Execute CLI context life cycle e2e tests for k8s target
e2e-context-k8s-tests:
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST} ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH} ; \
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GINKGO} --keep-going --output-dir ${ROOT_DIR}/test/e2e/testresults --json-report=results.json --keep-separate-reports --race --nodes=1 ${GOTEST_VERBOSE} -r ${ROOT_DIR}/test/e2e/context/k8s  --trace > /tmp/out && { cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; } || { exit_code=$$?; cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; exit $$exit_code; } \

.PHONY: e2e-airgapped-tests ## Execute CLI airgapped tests
e2e-airgapped-tests:
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_IMAGE_FOR_AIRGAPPED) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST} ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH} ; \
	export TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_IMAGE_FOR_AIRGAPPED) ; \
	export TANZU_CLI_E2E_AIRGAPPED_REPO=$(TANZU_CLI_E2E_AIRGAPPED_REPO) ; \
	export TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH=$(TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH) ; \
	export TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH_USERNAME=$(TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH_USERNAME) ; \
	export TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH_PASSWORD=$(TANZU_CLI_E2E_AIRGAPPED_REPO_WITH_AUTH_PASSWORD) ; \
	export TANZU_CLI_E2E_INPUT_CONFIG_DATA_FILE_PATH=$(TANZU_CLI_E2E_INPUT_CONFIG_DATA_FILE_PATH) ; \
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="Yes" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GINKGO} --keep-going --output-dir ${ROOT_DIR}/test/e2e/testresults --json-report=results.json --keep-separate-reports --race --nodes=1 ${GOTEST_VERBOSE} -r ${ROOT_DIR}/test/e2e/airgapped  --trace > /tmp/out && { cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; } || { exit_code=$$?; cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; exit $$exit_code; } \

.PHONY: e2e-catalog-tests ## Execute E2E catalog tests
e2e-catalog-tests:
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_PUBLIC_KEY_PATH) ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST} ; \
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH=${TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_CA_CERT_PATH} ; \
	export TANZU_CLI_PLUGIN_DISCOVERY_IMAGE_SIGNATURE_VERIFICATION_SKIP_LIST=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL) ; \
	export TANZU_CLI_CEIP_OPT_IN_PROMPT_ANSWER="No" ; \
	export TANZU_CLI_EULA_PROMPT_ANSWER="Yes" ; \
	${GINKGO} --keep-going --output-dir ${ROOT_DIR}/test/e2e/testresults --json-report=results.json --keep-separate-reports --race --nodes=1 ${GOTEST_VERBOSE} -r ${ROOT_DIR}/test/e2e/cataloge2e  --trace > /tmp/out && { cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; } || { exit_code=$$?; cat /tmp/out | grep -Ev 'STEP:|seconds|.go:'; rm /tmp/out; exit $$exit_code; } \

# Set the central repo to an inventory that has an extra column and then
# run the standard plugin life-cycle tests to make sure they still work
.PHONY: e2e-extra-column-tests ## Execute E2E tests with a DB with an extra column
e2e-extra-column-tests:
	export TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_URL=$(TANZU_CLI_E2E_TEST_LOCAL_CENTRAL_REPO_HOST)/tanzu-cli/plugins/extra:small ; \
	$(MAKE) e2e-plugin-lifecycle-tests

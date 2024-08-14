// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package csp

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strconv"

	"github.com/pkg/errors"
	"golang.org/x/term"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

const (
	// Tanzu CLI client ID that has http://127.0.0.1/callback as the
	// only allowed Redirect URI and does not have an associated client secret.
	tanzuCLIClientID     = "tanzu-cli-client-id"
	tanzuCLIClientSecret = ""
	defaultListenAddress = "127.0.0.1:0"
	defaultCallbackPath  = "/callback"

	centralConfigTanzuHubMetadata             = "cli.core.tanzu_hub_metadata"
	centralConfigTanzuDefaultCSPMetadata      = "cli.core.tanzu_default_csp_metadata"
	centralConfigTanzuTCSPMetadata            = "cli.core.tanzu_tcsp_metadata"
	centralConfigTanzuKnownIssersEndpoints    = "cli.core.tanzu_csp_known_issuer_endpoints"
	centralConfigCLIConfigCSPIssuerUpdateFlag = "cli.core.tanzu_cli_config_csp_issuer_update_flag"
	defaultCSPDisplayName                     = "Tanzu Platform"
	defaultCSPProductIdentifier               = "TANZU-SAAS"
)

// orgInfo to decode the CSP organization API response
type orgInfo struct {
	Name string `json:"displayName"`
}

type cspKnownIssuerEndpoints map[string]common.IssuerEndPoints

// TanzuCSPMetadata to parse the CSP metadata from central config
type TanzuCSPMetadata struct {
	IssuerProduction string `json:"issuerProduction" yaml:"issuerProduction"`
	IssuerStaging    string `json:"issuerStaging" yaml:"issuerStaging"`
}

func TanzuLogin(issuerURL string, opts ...common.LoginOption) (*common.Token, error) {
	cspKnownIssuerEPs := getCSPKnownIssuersEndpoints()
	h := common.NewTanzuLoginHandler(issuerURL, cspKnownIssuerEPs[issuerURL].AuthURL, cspKnownIssuerEPs[issuerURL].TokenURL, tanzuCLIClientID, tanzuCLIClientSecret, defaultListenAddress, defaultCallbackPath, config.CSPIdpType, GetOrgNameFromOrgID, nil, term.IsTerminal)
	for _, opt := range opts {
		if err := opt(h); err != nil {
			return nil, err
		}
	}

	return h.DoLogin()
}

// GetOrgNameFromOrgID fetches CSP Org Name given the Organization ID.
func GetOrgNameFromOrgID(orgID, accessToken, issuer string) (string, error) {
	apiURL := fmt.Sprintf("%s/orgs/%s", issuer, orgID)
	req, _ := http.NewRequestWithContext(context.Background(), "GET", apiURL, http.NoBody)
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := httpRestClient.Do(req)
	if err != nil {
		return "", errors.WithMessage(err, "failed to obtain the CSP organization information")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", errors.Errorf("failed to obtain the CSP organization information: %s", string(body))
	}

	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	org := orgInfo{}
	if err = json.Unmarshal(body, &org); err != nil {
		return "", errors.Wrap(err, "could not unmarshal CSP organization information")
	}

	return org.Name, nil
}

// GetCSPMetadata gets the CSP metadata from central config as best effort,
// If it fails to get the metadata from central config, it returns the default values
func GetCSPMetadata() TanzuCSPMetadata {
	cspMetaData := getCSPMetadataFromCentralConfig()
	// If failed to get the CSP Metadata from central config,
	// set the default Issuer URL of VCSP
	// TODO(prkalle): Update the default Issuers to TCSP issuer URL( If TCSP is not stable in current release, update it next release)
	//                This just defaults in the code, defaults in central config can be updated anytime
	if cspMetaData.IssuerStaging == "" {
		cspMetaData.IssuerStaging = StgIssuer
	}
	if cspMetaData.IssuerProduction == "" {
		cspMetaData.IssuerProduction = ProdIssuer
	}

	return cspMetaData
}

// getCSPMetadataFromCentralConfig gets the CSP metadata from central config as best effort
func getCSPMetadataFromCentralConfig() TanzuCSPMetadata {
	cspMetadata := TanzuCSPMetadata{}

	// Get the tanzu CSP metadata based on user preference
	useTanzuCSP, _ := strconv.ParseBool(os.Getenv(constants.UseTanzuCSP))
	if useTanzuCSP {
		_ = centralconfig.DefaultCentralConfigReader.GetCentralConfigEntry(centralConfigTanzuTCSPMetadata, &cspMetadata)
	} else {
		_ = centralconfig.DefaultCentralConfigReader.GetCentralConfigEntry(centralConfigTanzuDefaultCSPMetadata, &cspMetadata)
	}

	return cspMetadata
}

// getCSPKnownIssuersEndpoints gets the CSP known issuer endpoints from central config as best effort,
// If it fails to fetch data from central config, it returns the default values
func getCSPKnownIssuersEndpoints() cspKnownIssuerEndpoints {
	cspKnownIssuerEPs, err := getCSPKnownEndpointsFromCentralConfig()
	if err == nil {
		return cspKnownIssuerEPs
	}

	// If failed to get the CSP Known Issuer endpoints, use the defaults
	for key := range DefaultKnownIssuers {
		cspKnownIssuerEPs[key] = common.IssuerEndPoints{
			AuthURL:  DefaultKnownIssuers[key].AuthURL,
			TokenURL: DefaultKnownIssuers[key].TokenURL,
		}
	}

	return cspKnownIssuerEPs
}

// getCSPKnownEndpointsFromCentralConfig gets the CSP known endpoints from central config as best effort
func getCSPKnownEndpointsFromCentralConfig() (cspKnownIssuerEndpoints, error) {
	cspKnownIssuerEPs := cspKnownIssuerEndpoints{}

	err := centralconfig.DefaultCentralConfigReader.GetCentralConfigEntry(centralConfigTanzuKnownIssersEndpoints, &cspKnownIssuerEPs)
	if err != nil {
		return cspKnownIssuerEPs, err
	}

	return cspKnownIssuerEPs, nil
}

// GetIssuerUpdateFlagFromCentralConfig gets the issuer update flag (used to update the CLI config file)
// from Central config as best effort
func GetIssuerUpdateFlagFromCentralConfig() bool {
	updateIssuerInCLIConfig := false

	err := centralconfig.DefaultCentralConfigReader.GetCentralConfigEntry(centralConfigCLIConfigCSPIssuerUpdateFlag, &updateIssuerInCLIConfig)
	if err != nil {
		return updateIssuerInCLIConfig
	}

	return updateIssuerInCLIConfig
}

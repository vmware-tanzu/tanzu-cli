// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"regexp"
	"strings"

	"github.com/vmware-tanzu/tanzu-cli/pkg/centralconfig"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

func configureTanzuPlatformServiceEndpoints(tpEndpoint string) error {
	// Check and configure service endpoints through environment variable if configured
	// If it returns true, meaning all service endpoints are already configured, skip
	// next steps of configuring service endpoints from tpEndpoint.
	if configureServiceEndpointsIfNotConfigured(
		os.Getenv(constants.TPHubEndpoint),
		os.Getenv(constants.TPKubernetesOpsEndpoint),
		os.Getenv(constants.TPUCPEndpoint)) {
		return nil
	}

	if tpEndpoint == "" {
		return errors.New("invalid endpoint")
	}

	// If the url scheme is not provided by the user, default to using `https`
	if !strings.HasPrefix(tpEndpoint, "http://") && !strings.HasPrefix(tpEndpoint, "https://") {
		tpEndpoint = "https://" + tpEndpoint
	}
	if isTanzuPlatformSaaSEndpoint(tpEndpoint) {
		return configureTanzuPlatformServiceEndpointsForSaas(tpEndpoint)
	}
	return configureTanzuPlatformServiceEndpointsForSM(tpEndpoint)
}

func configureTanzuPlatformServiceEndpointsForSaas(tpEndpoint string) error {
	// Check if the custom endpoint mapping is available for the specified endpoint
	// If custom mapping is available configure the service endpoints and return
	endpointToServiceEndpointMap, _ := centralconfig.DefaultCentralConfigReader.GetTanzuPlatformEndpointToServiceEndpointMap()
	for endpoint, serviceEndpoints := range endpointToServiceEndpointMap {
		if strings.EqualFold(tpEndpoint, endpoint) {
			configureServiceEndpointsIfNotConfigured(serviceEndpoints.HubEndpoint, serviceEndpoints.TMCEndpoint, serviceEndpoints.UCPEndpoint)
			return nil
		}
	}

	// If custom mapping is not available for the endpoint,
	// use default fallback algorithm for the SaaS endpoints
	u, err := url.Parse(tpEndpoint)
	if err != nil {
		return err
	}

	if u.Host == "" {
		return fmt.Errorf("invalid URL: %s", tpEndpoint)
	}

	// If host starts with `www.` then it might because user might have specified the
	// tanzu platform url used for UI directly. In this can remove `www.` prefix.
	u.Host = strings.TrimPrefix(u.Host, "www.")

	configureServiceEndpointsIfNotConfigured(
		fmt.Sprintf("%s://api.%s", u.Scheme, u.Host),
		fmt.Sprintf("%s://ops.%s", u.Scheme, u.Host),
		fmt.Sprintf("%s://ucp.%s", u.Scheme, u.Host),
	)

	return nil
}

// configures the service endpoints for the SM deployment
// TODO(anuj/vui): Update once the SM algorithm is finalized
func configureTanzuPlatformServiceEndpointsForSM(tpEndpoint string) error {
	u, err := url.Parse(tpEndpoint)
	if err != nil {
		return err
	}
	if u.Scheme == "" {
		u.Scheme = "https"
	}
	if u.Host == "" {
		return fmt.Errorf("invalid URL: %s", tpEndpoint)
	}

	// If host starts with `www.` then it might because user might have specified the
	// tanzu platform url used for UI directly. In this can remove `www.` prefix.
	u.Host = strings.TrimPrefix(u.Host, "www.")

	tanzuHubEndpoint = fmt.Sprintf("%s://%s/hub", u.Scheme, u.Host)
	tanzuTMCEndpoint = fmt.Sprintf("%s://%s", u.Scheme, u.Host)
	tanzuUCPEndpoint = fmt.Sprintf("%s://%s/ucp", u.Scheme, u.Host)
	tanzuAuthEndpoint = fmt.Sprintf("%s://%s/auth", u.Scheme, u.Host)

	return nil
}

func isTanzuPlatformSaaSEndpoint(tpEndpoint string) bool {
	if forceCSP {
		return true
	}

	if tpEndpoint == "" {
		return false
	}

	saasEndpointRegularExpressions := centralconfig.DefaultCentralConfigReader.GetTanzuPlatformSaaSEndpointList()
	for _, endpointRegex := range saasEndpointRegularExpressions {
		// Create a regular expression pattern
		re, err := regexp.Compile(endpointRegex)
		if err != nil {
			continue
		}
		// If match found return true considering specified
		// tpEndpoint is SaaS endpoint
		if re.MatchString(tpEndpoint) {
			return true
		}
	}
	return false
}

// configureServiceEndpointsIfNotConfigured configures the service endpoints if one is not already configured
// It will return true if all service endpoints are configured, otherwise it will return false
func configureServiceEndpointsIfNotConfigured(hubEndpoint, tmcEndpoint, ucpEndpoint string) bool {
	if tanzuHubEndpoint == "" && hubEndpoint != "" {
		tanzuHubEndpoint = hubEndpoint + "/hub"
	}
	if tanzuUCPEndpoint == "" {
		tanzuUCPEndpoint = ucpEndpoint
	}
	if tanzuTMCEndpoint == "" {
		tanzuTMCEndpoint = tmcEndpoint
	}

	return (tanzuHubEndpoint != "" && tanzuTMCEndpoint != "" && tanzuUCPEndpoint != "")
}

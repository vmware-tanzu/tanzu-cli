// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package pluginmanager

import (
	"fmt"
	"os"
	"strings"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/discovery"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

const True = "true"
const HTTPS = "https"
const HTTP = "http"

func defaultDiscoverySourceBasedOnServer(server *configtypes.Server) []discovery.Discovery { //nolint:staticcheck // Deprecated
	var defaultDiscoveries []discovery.Discovery
	// If current server type is management-cluster, then add
	// the default kubernetes discovery endpoint pointing to the
	// management-cluster kubeconfig
	if server != nil && server.Type == configtypes.ManagementClusterServerType && server.ManagementClusterOpts != nil { //nolint:staticcheck // Deprecated
		defaultDiscoveries = append(defaultDiscoveries, defaultDiscoverySourceForK8sTargetedContext(server.Name, server.ManagementClusterOpts.Path, server.ManagementClusterOpts.Context))
	}
	return defaultDiscoveries
}

func defaultDiscoverySourceBasedOnContext(context *configtypes.Context) []discovery.Discovery {
	var defaultDiscoveries []discovery.Discovery

	// If current context is of type k8s, then add the default
	// kubernetes discovery endpoint pointing to the cluster kubeconfig
	// If the current context is of type tmc, then add the default REST endpoint
	// for the tmc discovery service
	if context.ContextType == configtypes.ContextTypeK8s && context.ClusterOpts != nil {
		defaultDiscoveries = append(defaultDiscoveries, defaultDiscoverySourceForK8sTargetedContext(context.Name, context.ClusterOpts.Path, context.ClusterOpts.Context))
	} else if context.ContextType == configtypes.ContextTypeTMC && context.GlobalOpts != nil {
		defaultDiscoveries = append(defaultDiscoveries, defaultDiscoverySourceForTMCTargetedContext(context))
	} else if context.ContextType == configtypes.ContextTypeTanzu && config.IsFeatureActivated(constants.FeatureContextScopedPluginDiscoveryForTanzuContext) {
		discObj, err := defaultDiscoverySourceForTanzuTargetedContext(context.Name)
		if err != nil {
			log.V(6).Infof("error while getting default discovery for context %q, error: %s", context.Name, err.Error())
		} else {
			defaultDiscoveries = append(defaultDiscoveries, discObj)
		}
	}

	return defaultDiscoveries
}

func defaultDiscoverySourceForK8sTargetedContext(tzContextName, kubeconfig, context string) discovery.Discovery {
	return discovery.NewKubernetesDiscovery(fmt.Sprintf("default-%s", tzContextName), kubeconfig, context, nil)
}

func defaultDiscoverySourceForTMCTargetedContext(context *configtypes.Context) discovery.Discovery {
	return discovery.NewRESTDiscovery(fmt.Sprintf("default-%s", context.Name), appendURLScheme(context.GlobalOpts.Endpoint), "v1alpha1/system/binaries/plugins")
}

func defaultDiscoverySourceForTanzuTargetedContext(tzContextName string) (discovery.Discovery, error) {
	tanzuContextDiscoveryEndpointPath := strings.TrimSpace(os.Getenv(constants.TanzuContextPluginDiscoveryPath))
	if tanzuContextDiscoveryEndpointPath == "" {
		tanzuContextDiscoveryEndpointPath = constants.TanzuContextPluginDiscoveryEndpointPath
	}

	kubeconfigBytes, err := config.GetKubeconfigForContext(tzContextName, config.ForCustomPath(tanzuContextDiscoveryEndpointPath))
	if err != nil {
		return nil, err
	}

	return discovery.NewKubernetesDiscovery(fmt.Sprintf("default-%s", tzContextName), "", "", kubeconfigBytes), nil
}

func appendURLScheme(endpoint string) string {
	// At present, the e2e test environment lacks support for HTTPS, thus hardcoding HTTPS is being skipped.
	if os.Getenv(constants.E2ETestEnvironment) == True {
		return endpoint
	}
	urlSec := strings.Split(endpoint, ":")
	// url does not have any scheme
	if len(urlSec) == 1 {
		return fmt.Sprintf("%s://%s", HTTPS, urlSec[0])
	} else if urlSec[0] == HTTPS || urlSec[0] == HTTP { // url starts with http/https scheme, do nothing
		return endpoint
	} else { // endpoint does not start http/https
		return fmt.Sprintf("%s://%s", HTTPS, endpoint)
	}
}

// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"os"
	"path/filepath"
	"time"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

// CheckDiscoveryName returns true if discovery name exists else return false
func CheckDiscoveryName(ds configtypes.PluginDiscovery, dn string) bool {
	return (ds.Kubernetes != nil && ds.Kubernetes.Name == dn) ||
		(ds.Local != nil && ds.Local.Name == dn) ||
		(ds.REST != nil && ds.REST.Name == dn) ||
		(ds.OCI != nil && ds.OCI.Name == dn)
}

// CompareDiscoverySource returns true if both discovery source are same for the given type
func CompareDiscoverySource(ds1, ds2 configtypes.PluginDiscovery, dsType string) bool {
	switch dsType {
	case common.DiscoveryTypeLocal:
		return compareLocalDiscoverySources(ds1, ds2)

	case common.DiscoveryTypeOCI:
		return compareOCIDiscoverySources(ds1, ds2)

	case common.DiscoveryTypeKubernetes:
		return compareK8sDiscoverySources(ds1, ds2)

	case common.DiscoveryTypeREST:
		return compareRESTDiscoverySources(ds1, ds2)
	}
	return false
}

func compareLocalDiscoverySources(ds1, ds2 configtypes.PluginDiscovery) bool {
	return ds1.Local != nil && ds2.Local != nil &&
		ds1.Local.Name == ds2.Local.Name &&
		ds1.Local.Path == ds2.Local.Path
}

func compareOCIDiscoverySources(ds1, ds2 configtypes.PluginDiscovery) bool {
	return ds1.OCI != nil && ds2.OCI != nil &&
		ds1.OCI.Name == ds2.OCI.Name &&
		ds1.OCI.Image == ds2.OCI.Image
}

func compareK8sDiscoverySources(ds1, ds2 configtypes.PluginDiscovery) bool {
	return ds1.Kubernetes != nil && ds2.Kubernetes != nil &&
		ds1.Kubernetes.Name == ds2.Kubernetes.Name &&
		ds1.Kubernetes.Path == ds2.Kubernetes.Path &&
		ds1.Kubernetes.Context == ds2.Kubernetes.Context
}

func compareRESTDiscoverySources(ds1, ds2 configtypes.PluginDiscovery) bool {
	return ds1.REST != nil && ds2.REST != nil &&
		ds1.REST.Name == ds2.REST.Name &&
		ds1.REST.BasePath == ds2.REST.BasePath &&
		ds1.REST.Endpoint == ds2.REST.Endpoint
}

func getDiscoverySourceNameAndURL(source configtypes.PluginDiscovery) (string, string, error) {
	var name string
	var url string
	switch {
	case source.OCI != nil:
		name = source.OCI.Name
		url = source.OCI.Image
	default:
		return "", "", errors.New("unknown discovery source type")
	}
	return name, url, nil
}

// RefreshDatabase function refreshes the plugin inventory database if the digest timestamp is past 24 hours
func RefreshDatabase() error {
	// Initialize digestExpirationThreshold with the default value
	digestExpirationThreshold := constants.DefaultPluginDBCacheRefreshThreshold

	// Check if the user has set a custom value for digest expiration threshold
	if envValue, ok := os.LookupEnv(constants.ConfigVariablePluginDBCacheRefreshThreshold); ok {
		if customThreshold, err := time.ParseDuration(envValue); err == nil {
			// Use the custom value if it's a valid duration
			digestExpirationThreshold = customThreshold
		}
	}

	// Fetch all Discovery sources
	sources, err := config.GetCLIDiscoverySources()
	if err != nil {
		return errors.Wrap(err, "failed to get discovery sources")
	}

	// Loop through each discovery source and refresh the db cached based on the digest expiry
	for _, source := range sources {
		// Get discovery source name and url
		name, _, err := getDiscoverySourceNameAndURL(source)
		if err != nil {
			return err
		}

		// Construct the digest file path
		pluginDataDir := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName, name)
		matches, _ := filepath.Glob(filepath.Join(pluginDataDir, "digest.*"))

		// If digest file is found
		if len(matches) == 1 {
			// check the modification time of the digest file to see
			// if the TTL is expired.
			if stat, err := os.Stat(matches[0]); err == nil {
				// Check if the digest timestamp is passed 24 hours if so refresh the database cache
				if time.Since(stat.ModTime()) > digestExpirationThreshold {
					if discObject, err := CreateDiscoveryFromV1alpha1(source); err == nil {
						_, _ = discObject.List()
					}
				}
			}
		} else {
			// digest file not found so refresh the database cache
			if discObject, err := CreateDiscoveryFromV1alpha1(source); err == nil {
				_, _ = discObject.List()
			}
		}
	}
	return nil
}

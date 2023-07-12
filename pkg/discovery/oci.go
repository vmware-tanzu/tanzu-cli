// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
	apimachineryjson "k8s.io/apimachinery/pkg/runtime/serializer/json"

	cliv1alpha1 "github.com/vmware-tanzu/tanzu-cli/apis/cli/v1alpha1"
	"github.com/vmware-tanzu/tanzu-cli/pkg/carvelhelpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugininventory"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

// OCIDiscovery is an artifact discovery endpoint utilizing OCI image
type OCIDiscovery struct {
	// name is a name of the discovery
	name string
	// image is an OCI compliant image. Which include DNS-compatible registry name,
	// a valid URI path(MAY contain zero or more ‘/’) and a valid tag
	// E.g., harbor.my-domain.local/tanzu-cli/plugins-manifest:latest
	// Contains a directory containing YAML files, each of which contains single
	// CLIPlugin API resource.
	image string
}

// NewOCIDiscovery returns a new Discovery using the specified OCI image.
func NewOCIDiscovery(name, image string, options ...DiscoveryOptions) Discovery {
	// Initialize discovery options
	opts := NewDiscoveryOpts()
	for _, option := range options {
		option(opts)
	}

	if !config.IsFeatureActivated(constants.FeatureDisableCentralRepositoryForTesting) {
		discovery := newDBBackedOCIDiscovery(name, image)
		discovery.pluginCriteria = opts.PluginDiscoveryCriteria
		discovery.useLocalCacheOnly = opts.UseLocalCacheOnly
		return discovery
	}

	return &OCIDiscovery{
		name:  name,
		image: image,
	}
}

// NewOCIGroupDiscovery returns a new plugn group Discovery using the specified OCI image.
func NewOCIGroupDiscovery(name, image string, options ...DiscoveryOptions) GroupDiscovery {
	// Initialize discovery options
	opts := NewDiscoveryOpts()
	for _, option := range options {
		option(opts)
	}

	discovery := newDBBackedOCIDiscovery(name, image)
	discovery.groupCriteria = opts.GroupDiscoveryCriteria
	discovery.useLocalCacheOnly = opts.UseLocalCacheOnly

	return discovery
}

func newDBBackedOCIDiscovery(name, image string) *DBBackedOCIDiscovery {
	// The plugin inventory uses relative image URIs to be future-proof.
	// Determine the image prefix from the main image.
	// E.g., if the main image is at project.registry.vmware.com/tanzu-cli/plugins/plugin-inventory:latest
	// then the image prefix should be project.registry.vmware.com/tanzu-cli/plugins/
	imagePrefix := path.Dir(image)
	// The data for the inventory is stored in the cache
	pluginDataDir := filepath.Join(common.DefaultCacheDir, common.PluginInventoryDirName, name)

	inventory := plugininventory.NewSQLiteInventory(filepath.Join(pluginDataDir, plugininventory.SQliteDBFileName), imagePrefix)
	return &DBBackedOCIDiscovery{
		name:          name,
		image:         image,
		pluginDataDir: pluginDataDir,
		inventory:     inventory,
	}
}

// List available plugins.
func (od *OCIDiscovery) List(_ ...PluginDiscoveryOptions) (plugins []Discovered, err error) {
	return od.Manifest()
}

// Name of the repository.
func (od *OCIDiscovery) Name() string {
	return od.name
}

// Type of the discovery.
func (od *OCIDiscovery) Type() string {
	return common.DiscoveryTypeOCI
}

// Manifest returns the manifest for a local repository.
func (od *OCIDiscovery) Manifest() ([]Discovered, error) {
	outputData, err := carvelhelpers.ProcessCarvelPackage(od.image)
	if err != nil {
		return nil, errors.Wrap(err, "error while processing package")
	}

	return processDiscoveryManifestData(outputData, od.name)
}

func processDiscoveryManifestData(data []byte, discoveryName string) ([]Discovered, error) {
	plugins := make([]Discovered, 0)

	for _, resourceYAML := range strings.Split(string(data), "---") {
		scheme, err := cliv1alpha1.SchemeBuilder.Build()
		if err != nil {
			return nil, errors.Wrap(err, "failed to create scheme")
		}
		s := apimachineryjson.NewSerializerWithOptions(apimachineryjson.DefaultMetaFactory, scheme, scheme,
			apimachineryjson.SerializerOptions{Yaml: true, Pretty: false, Strict: false})
		var p cliv1alpha1.CLIPlugin
		_, _, err = s.Decode([]byte(resourceYAML), nil, &p)
		if err != nil {
			return nil, errors.Wrap(err, "could not decode discovery manifests")
		}

		dp, err := DiscoveredFromK8sV1alpha1(&p)
		if err != nil {
			return nil, err
		}
		dp.Source = discoveryName
		dp.DiscoveryType = common.DiscoveryTypeOCI
		if dp.Name != "" {
			plugins = append(plugins, dp)
		}
	}
	return plugins, nil
}

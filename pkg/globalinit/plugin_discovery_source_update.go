// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package globalinit

import (
	"fmt"
	"io"

	cliconfig "github.com/vmware-tanzu/tanzu-cli/pkg/config"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/lastversion"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
)

// This global initializer checks if the last executed CLI version is < 1.3.0.
// If so it tries to update the plugin discovery source if needed

func init() {
	RegisterInitializer("Plugin Discovery Source Updater", triggerForPluginDiscoverySourceUpdater, updatePluginDiscoverySource)
}

func triggerForPluginDiscoverySourceUpdater() bool {
	// If the last executed CLI version is < 1.3.0, we need to update the discovery image source
	return lastversion.GetLastExecutedCLIVersion() == lastversion.OlderThan1_3_0
}

// updatePluginDiscoverySource updates the plugin discovery source to point CLI to
// new `projects.packages.broadcom.com` and also clears the EULA acceptance state
// stored in the CLI config file. So that EULA prompt is shown to the user again.
func updatePluginDiscoverySource(outStream io.Writer) error {
	ds, err := config.GetCLIDiscoverySource(cliconfig.DefaultStandaloneDiscoveryName)
	if err != nil {
		return err
	}

	if ds == nil || ds.OCI == nil || ds.OCI.Image != constants.TanzuCLIOldCentralPluginDiscoveryImage {
		// User must have manually updated discovery source. So do not overwrite that change.
		return nil
	}

	// Considering we are updating the discovery source to point to new registry,
	// User must be prompted to access the EULA again. So, let's first unset EULA status
	err = config.SetEULAStatus(config.EULAStatusUnset)
	if err != nil {
		return err
	}

	// Update the `default` discovery source to point to the new central plugin discovery image
	// Note: This update only modifies the discovery source and does not trigger a database refresh.
	// This is intentional, as we want to avoid downloading the image without the user re-accepting the EULA.
	// The database will be refreshed automatically by the CLI as part of its periodic refresh process.
	ds.OCI.Image = constants.TanzuCLIDefaultCentralPluginDiscoveryImage
	fmt.Fprintf(outStream, "Updating default plugin discovery source to %q...\n", constants.TanzuCLIDefaultCentralPluginDiscoveryImage)
	return config.SetCLIDiscoverySource(*ds)
}

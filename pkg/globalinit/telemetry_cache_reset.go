// Copyright 2025 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package globalinit

import (
	"io"

	"github.com/vmware-tanzu/tanzu-cli/pkg/lastversion"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugincmdtree"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// This global initializer checks if the last executed CLI version is < 1.5.3.
// If so it removes the plugin command-tree cache file used by telemetry.
// This allows a fresh cache to be gradually built using the latest command tree format.
// Note that this initializer does not build the cache, it only removes the existing cache file.

func init() {
	RegisterInitializer("Plugin Command-Tree Reset", triggerForPluginCmdTreeReset, resetPluginCmdTree)
}

func triggerForPluginCmdTreeReset() bool {
	// If the last executed CLI version is < 1.5.3, we need to remove the command-tree cache file.
	return lastversion.IsLessThan(lastversion.Version1_5_3)
}

// resetPluginCmdTree deletes the plugin command-tree cache.
func resetPluginCmdTree(_ io.Writer) error {
	pct, err := plugincmdtree.NewCache()
	if err != nil {
		return err
	}

	log.V(7).Infof("Clearing telemetry plugin command-tree cache for all plugins...")
	return pct.DeleteTree()
}

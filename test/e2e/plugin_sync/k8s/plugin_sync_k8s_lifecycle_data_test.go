// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package pluginsynce2ek8s

import "github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"

var PluginsMultiVersionInstallTests = []struct {
	pluginInfo               framework.PluginInfo
	expectedInstalledVersion string
	err                      string
}{
	{framework.PluginInfo{Name: "package", Target: "kubernetes", Version: "v1", Description: "package functionality"}, "v1.11.3", ""},
	{framework.PluginInfo{Name: "package", Target: "kubernetes", Version: "v1.9", Description: "package functionality"}, "v1.9.2-beta.1", ""},
	{framework.PluginInfo{Name: "package", Target: "kubernetes", Version: "v1.10", Description: "package functionality"}, "v1.10.2", ""},
	{framework.PluginInfo{Name: "package", Target: "kubernetes", Version: "v1.11.2", Description: "package functionality"}, "v1.11.2", ""},
	{framework.PluginInfo{Name: "package", Target: "kubernetes", Version: "v1.12", Description: "package functionality"}, "", "unable to find plugin 'package' with version 'v1.12' for target 'kubernetes'"},
	{framework.PluginInfo{Name: "package", Target: "kubernetes", Version: "v2", Description: "package functionality"}, "v2.3.5", ""},
	{framework.PluginInfo{Name: "package", Target: "kubernetes", Version: "v2.3.0", Description: "package functionality"}, "v2.3.0", ""},
}

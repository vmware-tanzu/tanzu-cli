// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package cataloge2e implements e2e tests specific to catalog cache updates
package cataloge2e

import (
	"sync"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/vmware-tanzu/tanzu-cli/test/e2e/framework"
)

var tf *framework.Framework

var _ = framework.CLICoreDescribe("[Tests:E2E][Feature:Catalog-Update]", func() {

	Context("Test catalog update in parallel when 2 telemetry plugins are present", func() {
		It("should return correct # of plugins after parallel execution and should not corrupt the catalog", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")

			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(BeNumerically("==", 0), "plugins list should not return any plugins after plugin clean")

			// Install "vmware-tkg/default" plugin group because it contains
			// telemetry plugin with kubernetes target
			err = tf.PluginCmd.InstallPluginsFromGroup("", "vmware-tkg/default")
			Expect(err).To(BeNil(), "should not get any error for installing all plugins from plugin group")

			pluginsList, err = framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(BeNumerically("==", 8), "plugins list should return 8 plugins installing plugin group")

			// Run the tanzu version command in parallel
			// Run 50 commands in parallel total 10 times
			for i := 0; i < 10; i++ {
				var wg sync.WaitGroup
				for i := 0; i < 50; i++ {
					wg.Add(1)
					go func() {
						defer wg.Done()
						_, _ = tf.CliOps.CLIVersion()
					}()
				}
				wg.Wait()
			}

			// After running above commands in parallel check the plugin list
			// and it should still return the correct plugin count
			pluginsList, err = framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(BeNumerically("==", 8), "plugins list should return 8 plugins installing plugin group")
		})
	})

	Context("Test catalog update in parallel by installing multiple plugins at a time", func() {
		It("should return correct # of plugins after parallel execution and should not corrupt the catalog", func() {
			err := tf.PluginCmd.CleanPlugins()
			Expect(err).To(BeNil(), "should not get any error for plugin clean")

			pluginsList, err := framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(BeNumerically("==", 0), "plugins list should not return any plugins after plugin clean")

			// Install all plugins from the "vmware-tkg/default" plugin group individually in parallel
			pluginGroupGet, err := tf.PluginCmd.GetPluginGroup("vmware-tkg/default", "")
			Expect(err).To(BeNil(), "should not get any error for installing all plugins from plugin group")

			var wg sync.WaitGroup
			for i := range pluginGroupGet {
				wg.Add(1)
				go func(pluginGroup *framework.PluginGroupGet) {
					defer wg.Done()
					_ = tf.PluginCmd.InstallPlugin(pluginGroup.PluginName, pluginGroup.PluginTarget, pluginGroup.PluginVersion)
				}(pluginGroupGet[i])
			}
			wg.Wait()

			// After running above commands in parallel check the plugin list
			// and it should still return the correct plugin count
			pluginsList, err = framework.GetPluginsList(tf, true)
			Expect(err).To(BeNil(), "should not get any error for plugin list")
			Expect(len(pluginsList)).Should(BeNumerically("==", len(pluginGroupGet)+1), "plugins list should return all plugins in plugin group + telemetry essential plugin")
		})
	})
})

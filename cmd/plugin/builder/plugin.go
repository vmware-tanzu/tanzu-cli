// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package main

import (
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/command"
	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/imgpkg"
	"github.com/vmware-tanzu/tanzu-cli/cmd/plugin/builder/plugin"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
)

// NewPluginCmd creates a new command for plugin operations.
func NewPluginCmd() *cobra.Command {
	var pluginCmd = &cobra.Command{
		Use:   "plugin",
		Short: "Plugin Operations",
	}

	pluginCmd.SetUsageFunc(cli.SubCmdUsageFunc)

	pluginCmd.AddCommand(
		newPluginBuildCmd(),
		newPluginBuildPackageCmd(),
		newPluginPublishPackageCmd(),
	)
	return pluginCmd
}

type pluginBuildFlags struct {
	PluginDir                  string
	ArtifactDir                string
	LDFlags                    string
	OSArch                     []string
	Version                    string
	Match                      string
	PluginScopeAssociationFile string
}

type pluginBuildPackageFlags struct {
	BinaryArtifactDir  string
	PackageArtifactDir string
	LocalOCIRepository string
}

type pluginPublishPackageFlags struct {
	PackageArtifactDir string
	Repository         string
	Publisher          string
	Vendor             string
	DryRun             bool
}

func newPluginBuildCmd() *cobra.Command {
	var pbFlags = &pluginBuildFlags{}

	var pluginBuildCmd = &cobra.Command{
		Use:          "build",
		Short:        "Build plugins",
		SilenceUsage: true,
		Example: `# Build all plugins under 'cmd/plugin' directory for local host os and arch
  tanzu builder plugin build --path ./cmd/plugin --version v0.0.2 --os-arch local

  # Build all plugins under 'cmd/plugin' directory for os-arch 'darwin_amd64', 'linux_amd64', 'windows_amd64'
  tanzu builder plugin build --path ./cmd/plugin --version v0.0.2 --os-arch darwin_amd64 --os-arch linux_amd64 --os-arch windows_amd64

  # Build only foo plugin under 'cmd/plugin' directory for all supported os-arch
  tanzu builder plugin build --path ./cmd/plugin --version v0.0.2 --os-arch all --match foo`,
		RunE: func(cmd *cobra.Command, args []string) error {
			compileArgs := &command.PluginCompileArgs{
				Match:                      pbFlags.Match,
				TargetArch:                 pbFlags.OSArch,
				SourcePath:                 pbFlags.PluginDir,
				ArtifactsDir:               pbFlags.ArtifactDir,
				LDFlags:                    pbFlags.LDFlags,
				Version:                    pbFlags.Version,
				PluginScopeAssociationFile: pbFlags.PluginScopeAssociationFile,
				GroupByOSArch:              true,
			}

			return command.Compile(compileArgs)
		},
	}

	pluginBuildCmd.Flags().StringVarP(&pbFlags.PluginDir, "path", "", "./cmd/plugin", "path of plugin directory")
	pluginBuildCmd.Flags().StringVarP(&pbFlags.ArtifactDir, "binary-artifacts", "", "./artifacts", "path to output artifacts directory")
	pluginBuildCmd.Flags().StringVarP(&pbFlags.LDFlags, "ldflags", "", "", "ldflags to set on build")
	pluginBuildCmd.Flags().StringArrayVarP(&pbFlags.OSArch, "os-arch", "", []string{"all"}, "compile for specific os-arch, use 'local' for host os, use '<os>_<arch>' for specific")
	pluginBuildCmd.Flags().StringVarP(&pbFlags.Version, "version", "v", "", "version of the plugins")
	pluginBuildCmd.Flags().StringVarP(&pbFlags.Match, "match", "", "*", "match a plugin name to build, supports globbing")
	pluginBuildCmd.Flags().StringVarP(&pbFlags.PluginScopeAssociationFile, "plugin-scope-association-file", "", "", "file specifying plugin scope association")

	_ = pluginBuildCmd.MarkFlagRequired("version")

	return pluginBuildCmd
}

func newPluginBuildPackageCmd() *cobra.Command {
	var pbpFlags = &pluginBuildPackageFlags{}

	var pluginBuildPackageCmd = &cobra.Command{
		Use:          "build-package",
		Short:        "Build plugin packages",
		Long:         "Build plugin packages OCI image as tar.gz file that can be published to any repository",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			bppArgs := &plugin.BuildPluginPackageOptions{
				BinaryArtifactDir:  pbpFlags.BinaryArtifactDir,
				PackageArtifactDir: pbpFlags.PackageArtifactDir,
				LocalOCIRegistry:   pbpFlags.LocalOCIRepository,
				ImgpkgOptions:      imgpkg.NewImgpkgCLIWrapper(),
			}
			return bppArgs.BuildPluginPackages()
		},
	}

	pluginBuildPackageCmd.Flags().StringVarP(&pbpFlags.BinaryArtifactDir, "binary-artifacts", "", "./artifacts/plugins", "plugin binary artifact directory")
	pluginBuildPackageCmd.Flags().StringVarP(&pbpFlags.PackageArtifactDir, "package-artifacts", "", "./artifacts/packages", "plugin package artifacts directory")
	pluginBuildPackageCmd.Flags().StringVarP(&pbpFlags.LocalOCIRepository, "oci-registry", "", "", "local oci-registry to use for generating packages")

	return pluginBuildPackageCmd
}

func newPluginPublishPackageCmd() *cobra.Command {
	var pppFlags = &pluginPublishPackageFlags{}

	var pluginBuildPackageCmd = &cobra.Command{
		Use:          "publish-package",
		Short:        "Publish plugin packages",
		Long:         "Publish plugin packages as OCI image to specified repository",
		SilenceUsage: true,
		RunE: func(cmd *cobra.Command, args []string) error {
			bppArgs := &plugin.PublishPluginPackageOptions{
				PackageArtifactDir: pppFlags.PackageArtifactDir,
				Publisher:          pppFlags.Publisher,
				Vendor:             pppFlags.Vendor,
				Repository:         pppFlags.Repository,
				DryRun:             pppFlags.DryRun,
				ImgpkgOptions:      imgpkg.NewImgpkgCLIWrapper(),
			}
			return bppArgs.PublishPluginPackages()
		},
	}

	pluginBuildPackageCmd.Flags().StringVarP(&pppFlags.PackageArtifactDir, "package-artifacts", "", "./artifacts/packages", "plugin package artifacts directory")
	pluginBuildPackageCmd.Flags().StringVarP(&pppFlags.Repository, "repository", "", "", "repository to publish plugins")
	pluginBuildPackageCmd.Flags().StringVarP(&pppFlags.Vendor, "vendor", "", "", "name of the vendor")
	pluginBuildPackageCmd.Flags().StringVarP(&pppFlags.Publisher, "publisher", "", "", "name of the publisher")
	pluginBuildPackageCmd.Flags().BoolVarP(&pppFlags.DryRun, "dry-run", "", false, "show commands without publishing plugin packages")

	_ = pluginBuildPackageCmd.MarkFlagRequired("repository")
	_ = pluginBuildPackageCmd.MarkFlagRequired("vendor")
	_ = pluginBuildPackageCmd.MarkFlagRequired("publisher")

	return pluginBuildPackageCmd
}

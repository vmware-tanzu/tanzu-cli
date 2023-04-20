// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package carvelhelpers

import (
	"os"
	"runtime"
	"strings"

	"github.com/pkg/errors"

	ctlimg "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"

	"github.com/vmware-tanzu/tanzu-cli/pkg/clientconfighelpers"
	"github.com/vmware-tanzu/tanzu-cli/pkg/configpaths"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/registry"
)

// GetFilesMapFromImage returns map of files metadata
// It takes os environment variables for custom repository and proxy
// configuration into account while downloading image from repository
func GetFilesMapFromImage(imageWithTag string) (map[string][]byte, error) {
	return NewImageOperationsImpl().GetFilesMapFromImage(imageWithTag)
}

// DownloadImageAndSaveFilesToDir reads a plain OCI image and saves its
// files to the specified location.
func DownloadImageAndSaveFilesToDir(imageWithTag, destinationDir string) error {
	return NewImageOperationsImpl().DownloadImageAndSaveFilesToDir(imageWithTag, destinationDir)
}

// GetImageDigest gets digest of the image
func GetImageDigest(imageWithTag string) (string, string, error) {
	return NewImageOperationsImpl().GetImageDigest(imageWithTag)
}

// newRegistry returns a new registry object by also
// taking into account for any custom registry or proxy
// environment variable provided by the user
func newRegistry() (registry.Registry, error) {
	verifyCerts := true
	skipVerifyCerts := os.Getenv(constants.ConfigVariableCustomImageRepositorySkipTLSVerify)
	if strings.EqualFold(skipVerifyCerts, "true") {
		verifyCerts = false
	}

	registryOpts := &ctlimg.Opts{
		VerifyCerts: verifyCerts,
		Anon:        true,
	}

	if runtime.GOOS == "windows" {
		err := clientconfighelpers.AddRegistryTrustedRootCertsFileForWindows(registryOpts)
		if err != nil {
			return nil, err
		}
	}

	caCertBytes, err := clientconfighelpers.GetCustomRepositoryCaCertificateForClient()
	if err == nil && len(caCertBytes) != 0 {
		filePath, err := configpaths.GetRegistryCertFile()
		if err != nil {
			return nil, err
		}
		err = os.WriteFile(filePath, caCertBytes, 0o644)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to write the custom image registry CA cert to file '%s'", filePath)
		}
		registryOpts.CACertPaths = append(registryOpts.CACertPaths, filePath)
	}

	return registry.New(registryOpts)
}

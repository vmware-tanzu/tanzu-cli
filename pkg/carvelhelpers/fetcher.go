// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package carvelhelpers

import (
	"os"
	"strings"

	"github.com/pkg/errors"

	ctlimg "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/registry"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
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

// newRegistry returns a new registry object by also taking
// into account for any custom registry provided by the user
func newRegistry(registryHost string) (registry.Registry, error) {
	registryOpts := &ctlimg.Opts{}

	authenticatedRegistries := strings.Split(os.Getenv(constants.AuthenticatedRegistry), ",")
	if !utils.ContainsRegistry(authenticatedRegistries, registryHost) {
		registryOpts.Anon = true
	}

	regCertOptions, err := registry.GetRegistryCertOptions(registryHost)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get the registry certificate configuration")
	}
	registryOpts.CACertPaths = regCertOptions.CACertPaths
	registryOpts.VerifyCerts = !(regCertOptions.SkipCertVerify)
	registryOpts.Insecure = regCertOptions.Insecure
	return registry.New(registryOpts)
}

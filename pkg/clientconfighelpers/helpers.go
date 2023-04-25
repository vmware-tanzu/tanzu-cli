// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package clientconfighelpers implements helper function
// related to client configurations
package clientconfighelpers

import (
	"encoding/base64"
	"os"

	ctlimg "github.com/vmware-tanzu/carvel-imgpkg/pkg/imgpkg/registry"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/configpaths"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

// GetCustomRepositoryCaCertificateForClient returns CA certificate to use with cli client
// This function reads the CA certificate from following variables in decreasing order of precedence:
// 1. PROXY_CA_CERT
// 2. TKG_PROXY_CA_CERT
// 3. TKG_CUSTOM_IMAGE_REPOSITORY_CA_CERTIFICATE
func GetCustomRepositoryCaCertificateForClient() ([]byte, error) {
	caCert := ""
	var errProxyCACert, errTkgProxyCACertValue, errCustomImageRepoCACert error
	var proxyCACertValue, tkgProxyCACertValue, customImageRepoCACert string

	// Get the proxy configuration from os environment variable
	proxyCACertValue = os.Getenv(constants.ProxyCACert)
	tkgProxyCACertValue = os.Getenv(constants.TKGProxyCACert)
	customImageRepoCACert = os.Getenv(constants.ConfigVariableCustomImageRepositoryCaCertificate)

	if errProxyCACert == nil && proxyCACertValue != "" {
		caCert = proxyCACertValue
	} else if errTkgProxyCACertValue == nil && tkgProxyCACertValue != "" {
		caCert = tkgProxyCACertValue
	} else if errCustomImageRepoCACert == nil && customImageRepoCACert != "" {
		caCert = customImageRepoCACert
	} else {
		// return empty content when none is specified
		return []byte{}, nil
	}

	decoded, err := base64.StdEncoding.DecodeString(caCert)
	if err != nil {
		return nil, errors.Wrap(err, "unable to decode the base64-encoded custom registry CA certificate string")
	}
	return decoded, nil
}

// AddRegistryTrustedRootCertsFileForWindows adds CA certificate to registry options for windows environments
func AddRegistryTrustedRootCertsFileForWindows(registryOpts *ctlimg.Opts) error {
	filePath, err := configpaths.GetRegistryTrustedCACertFileForWindows()
	if err != nil {
		return err
	}
	err = os.WriteFile(filePath, projectsRegistryCA, constants.ConfigFilePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to write the registry trusted CA cert to file '%s'", filePath)
	}
	registryOpts.CACertPaths = append(registryOpts.CACertPaths, filePath)
	return nil
}
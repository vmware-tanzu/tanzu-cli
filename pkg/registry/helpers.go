// Copyright 2022-23 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"encoding/base64"
	"os"
	"runtime"
	"strconv"

	regname "github.com/google/go-containerregistry/pkg/name"
	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/configpaths"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

type RegistryCertOptions struct {
	CACertPaths    []string
	SkipCertVerify bool
	Insecure       bool
}

func GetRegistryCertOptions(registryHost string) (*RegistryCertOptions, error) {
	registryCertOpts := &RegistryCertOptions{
		SkipCertVerify: false,
		Insecure:       false,
	}

	if runtime.GOOS == "windows" {
		err := AddRegistryTrustedRootCertsFileForWindows(registryCertOpts)
		if err != nil {
			return nil, err
		}
	}

	// check if the custom cert data is configured for the registry
	if exists, _ := configlib.CertExists(registryHost); !exists {
		return registryCertOpts, nil
	}
	cert, err := configlib.GetCert(registryHost)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to get the custom certificate configuration for host %q", registryHost)
	}

	err = updateRegistryCertOptions(cert, registryCertOpts)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to updated the registry cert options")
	}

	return registryCertOpts, nil
}

// updateRegistryCertOptions sets the registry options by taking the custom certificate data configured for registry as input
func updateRegistryCertOptions(cert *configtypes.Cert, registryCertOpts *RegistryCertOptions) error {
	if cert.SkipCertVerify != "" {
		skipVerifyCerts, _ := strconv.ParseBool(cert.SkipCertVerify)
		registryCertOpts.SkipCertVerify = skipVerifyCerts
	}
	if cert.Insecure != "" {
		insecure, _ := strconv.ParseBool(cert.Insecure)
		registryCertOpts.Insecure = insecure
	}

	if cert.CACertData != "" {
		caCertBytes, err := base64.StdEncoding.DecodeString(cert.CACertData)
		if err != nil {
			return errors.Wrap(err, "unable to decode the base64-encoded custom registry CA certificate string")
		}
		if len(caCertBytes) != 0 {
			filePath, err := configpaths.GetRegistryCertFile()
			if err != nil {
				return err
			}
			err = os.WriteFile(filePath, caCertBytes, 0o644)
			if err != nil {
				return errors.Wrapf(err, "failed to write the custom image registry CA cert to file '%s'", filePath)
			}
			registryCertOpts.CACertPaths = append(registryCertOpts.CACertPaths, filePath)
		}
	}
	return nil
}

// AddRegistryTrustedRootCertsFileForWindows adds CA certificate to registry options for Windows environments
func AddRegistryTrustedRootCertsFileForWindows(registryCertOpts *RegistryCertOptions) error {
	filePath, err := configpaths.GetRegistryTrustedCACertFileForWindows()
	if err != nil {
		return err
	}
	err = os.WriteFile(filePath, projectsRegistryCA, constants.ConfigFilePermissions)
	if err != nil {
		return errors.Wrapf(err, "failed to write the registry trusted CA cert to file '%s'", filePath)
	}
	registryCertOpts.CACertPaths = append(registryCertOpts.CACertPaths, filePath)
	return nil
}

// GetRegistryName extracts the registry name from the image name with/without image tag
// (e.g. localhost:9876/tanzu-cli/plugins/central:small => localhost:9876)
func GetRegistryName(imageName string) (string, error) {
	tag, err := regname.NewTag(imageName)
	if err != nil {
		return "", errors.Wrapf(err, "unable to fetch registry name from image %q", imageName)
	}
	return tag.Registry.Name(), nil
}

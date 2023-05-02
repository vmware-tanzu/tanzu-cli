// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package sigverifier implements helper functions to verify inventory image signature
package sigverifier

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	"github.com/vmware-tanzu/tanzu-cli/pkg/cosignhelper"
	"github.com/vmware-tanzu/tanzu-cli/pkg/registry"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

func VerifyInventoryImageSignature(image string) error {
	cosignVerifier, err := getCosignVerifier(image)
	if err != nil {
		return errors.Wrapf(err, "failed to initialize the cosign verifier")
	}

	if sigVerifyErr := verifyInventoryImageSignature(image, cosignVerifier); sigVerifyErr != nil {
		log.Warningf("Unable to verify the plugins discovery image signature: %v", sigVerifyErr)
		// TODO(pkalle): Update the message to convey user to check if they could use the latest public key after we get details of the well known location of the public key
		errMsg := fmt.Sprintf("Fatal, plugins discovery image signature verification failed. The `tanzu` CLI can not ensure the integrity of the plugins to be installed. To ignore this validation please append %q to the comma-separated list in the environment variable %q.  This is NOT RECOMMENDED and could put your environment at risk!",
			image, constants.PluginDiscoveryImageSignatureVerificationSkipList)
		log.Fatal(nil, errMsg)
	}
	return nil
}

func getCosignVerifier(image string) (cosignhelper.Cosignhelper, error) {
	// Get the custom public key path and prepare cosign verifier, if empty, cosign verifier would use embedded public key for verification
	customPublicKeyPath := os.Getenv(constants.PublicKeyPathForPluginDiscoveryImageSignature)

	registryOptions, err := getCosignVerifierRegistryOptions(image)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to prepare the registry options for cosign verification")
	}
	return cosignhelper.NewCosignVerifier(customPublicKeyPath, registryOptions), nil
}

// getCosignVerifierRegistryOptions prepares the registry options by including the custom certificate configuration if any
func getCosignVerifierRegistryOptions(image string) (*cosignhelper.RegistryOptions, error) {
	registryOpts := &cosignhelper.RegistryOptions{}
	registryName, err := registry.GetRegistryName(strings.TrimSpace(image))
	if err != nil {
		return nil, err
	}
	// get the certificate configuration and update the registry options
	regCertOptions, err := registry.GetRegistryCertOptions(registryName)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to get the registry certificate configuration")
	}
	registryOpts.CACertPaths = regCertOptions.CACertPaths
	registryOpts.SkipCertVerify = regCertOptions.SkipCertVerify
	registryOpts.AllowInsecure = regCertOptions.Insecure

	return registryOpts, nil
}

func verifyInventoryImageSignature(image string, verifier cosignhelper.Cosignhelper) error {
	signatureVerificationSkipSet := getPluginDiscoveryImagesSkippedForSignatureVerification()
	if _, exists := signatureVerificationSkipSet[strings.TrimSpace(image)]; exists {
		// log warning message iff user had not chosen to skip warning message for signature verification
		if skip, _ := strconv.ParseBool(os.Getenv(constants.SuppressSkipSignatureVerificationWarning)); !skip {
			log.Warningf("Skipping the plugins discovery image signature verification for %q\n ", image)
		}
		return nil
	}

	err := verifier.Verify(context.Background(), []string{image})
	if err != nil {
		return err
	}
	return nil
}

func getPluginDiscoveryImagesSkippedForSignatureVerification() map[string]struct{} {
	discoveryImages := map[string]struct{}{}
	discoveryImagesList := strings.Split(os.Getenv(constants.PluginDiscoveryImageSignatureVerificationSkipList), ",")
	for _, image := range discoveryImagesList {
		image = strings.TrimSpace(image)
		if image != "" {
			discoveryImages[image] = struct{}{}
		}
	}
	return discoveryImages
}

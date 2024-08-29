// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package centralconfig

var (
	// NOTE: This value will be overwritten from the value specified in the central configuration.
	// It serves as a fallback default only if reading the central configuration fails.
	DefaultTanzuPlatformEndpoint = "https://api.tanzu.cloud.vmware.com"

	defaultSaaSEndpoints = []string{
		"https://(www.)?platform(.)*.tanzu.broadcom.com",
		"https://api.tanzu(.)*.cloud.vmware.com",
	}

	// DefaultCentralConfigReader is a pre-initialized default central configuration reader instance.
	// This global object can be used directly to read configuration from the default central configuration.
	DefaultCentralConfigReader = newDefaultCentralConfigReader()
)

func init() {
	// initialize the value of `DefaultTanzuPlatformEndpoint` from default central configuration
	endpoint, _ := DefaultCentralConfigReader.GetDefaultTanzuEndpoint()
	if endpoint != "" {
		DefaultTanzuPlatformEndpoint = endpoint
	}
}

// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package centralconfig

// TanzuPlatformEndpointToServiceEndpointMap is a map that maps Tanzu platform endpoints to service endpoint maps.
// It is used to store the endpoints for different Tanzu services, such as UCP, TMC, and Hub.
type TanzuPlatformEndpointToServiceEndpointMap map[string]ServiceEndpointMap

// ServiceEndpointMap represents a map of service endpoints for a Tanzu platform.
// It contains the endpoints for UCP, TMC, and Hub services.
type ServiceEndpointMap struct {
	// UCPEndpoint is the endpoint for the UCP (Unified Control Plane) service.
	UCPEndpoint string `yaml:"ucp"`
	// TMCEndpoint is the endpoint for the TMC (Tanzu Mission Control) service.
	TMCEndpoint string `yaml:"tmc"`
	// HubEndpoint is the endpoint for the Tanzu Hub service.
	HubEndpoint string `yaml:"hub"`
}

// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package interfaces is collection of generic interfaces
package interfaces

import (
	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

//go:generate counterfeiter -o ../fakes/config_client_fake.go . ConfigClientWrapper
type ConfigClientWrapper interface {
	GetEnvConfigurations() map[string]string
	StoreClientConfig(clientConfig *configtypes.ClientConfig) error
	AcquireTanzuConfigLock()
	ReleaseTanzuConfigLock()
}

type configClientWrapperImpl struct{}

func NewConfigClient() ConfigClientWrapper {
	return &configClientWrapperImpl{}
}

func (cc *configClientWrapperImpl) GetEnvConfigurations() map[string]string {
	return config.GetEnvConfigurations()
}

func (cc *configClientWrapperImpl) AcquireTanzuConfigLock() {
	config.AcquireTanzuConfigLock()
}

func (cc *configClientWrapperImpl) ReleaseTanzuConfigLock() {
	config.ReleaseTanzuConfigLock()
}

func (cc *configClientWrapperImpl) StoreClientConfig(clientConfig *configtypes.ClientConfig) error {
	return config.StoreClientConfig(clientConfig)
}

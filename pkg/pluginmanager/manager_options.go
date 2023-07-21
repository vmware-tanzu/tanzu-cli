// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package pluginmanager

import (
	"os"
	"strconv"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// PluginManagerOpts options to customize plugin lifecycle operations
type PluginManagerOpts struct {
	showLogs bool // Enable or disable logs
}

// GetLogMode sets the log mode based on the environment variable.
func (p *PluginManagerOpts) GetLogMode() bool {
	return p.showLogs
}

// SetLogMode sets the log mode based on the environment variable.
func (p *PluginManagerOpts) SetLogMode() {
	// Check if env is set which takes precedence
	envLogMode, err := strconv.ParseBool(os.Getenv(constants.TanzuCLIShowPluginInstallationLogs))
	if err != nil {
		log.QuietMode(!p.showLogs)
		return
	}

	if envLogMode {
		log.QuietMode(false)
		p.showLogs = true
	} else {
		log.QuietMode(true)
		p.showLogs = false
	}
}

// ResetLogMode reset the log mode to show logs
func (p *PluginManagerOpts) ResetLogMode() {
	log.QuietMode(false)
	p.showLogs = true
}

type PluginManagerOptions func(p *PluginManagerOpts)

func DisableLogs() PluginManagerOptions {
	return func(p *PluginManagerOpts) {
		p.showLogs = false
	}
}

func EnableLogs() PluginManagerOptions {
	return func(p *PluginManagerOpts) {
		p.showLogs = true
	}
}

// NewPluginManagerOpts creates a new PluginManagerOpts instance with provided options.
func NewPluginManagerOpts(opts ...PluginManagerOptions) *PluginManagerOpts {
	// By default logs are enabled
	p := &PluginManagerOpts{
		showLogs: true,
	}

	for _, opt := range opts {
		opt(p)
	}

	return p
}

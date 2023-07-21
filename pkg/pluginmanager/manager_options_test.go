// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package pluginmanager

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
)

// Test cases
var getLogModeCases = []struct {
	name     string
	showLogs bool
	expected bool
}{
	{"ShowLogs is true", true, true},
	{"ShowLogs is false", false, false},
}

var setLogModeCases = []struct {
	name     string
	env      string
	showLogs bool
	expected bool
}{
	{"Environment variable is true", "true", true, true},
	{"Environment variable is false", "false", true, false},
	{"Environment variable is not set and enable logs", "", true, true},
	{"Environment variable is not set and disable logs", "", false, false},
}

var resetLogModeCases = []struct {
	name     string
	showLogs bool
	expected bool
}{
	{"Reset log mode", false, true},
}

// Tests
func TestGetLogMode(t *testing.T) {
	for _, tc := range getLogModeCases {
		t.Run(tc.name, func(t *testing.T) {
			var p *PluginManagerOpts
			if tc.showLogs {
				p = NewPluginManagerOpts()
			} else {
				p = NewPluginManagerOpts(DisableLogs())
			}
			assert.Equal(t, tc.expected, p.GetLogMode())
		})
	}
}

func TestSetLogMode(t *testing.T) {
	for _, tc := range setLogModeCases {
		t.Run(tc.name, func(t *testing.T) {
			os.Setenv(constants.TanzuCLIShowPluginInstallationLogs, tc.env)
			var p *PluginManagerOpts
			if tc.showLogs {
				p = NewPluginManagerOpts()
			} else {
				p = NewPluginManagerOpts(DisableLogs())
			}
			p.SetLogMode()
			assert.Equal(t, tc.expected, p.GetLogMode())
		})
	}
}

func TestResetLogMode(t *testing.T) {
	for _, tc := range resetLogModeCases {
		t.Run(tc.name, func(t *testing.T) {
			var p *PluginManagerOpts
			if tc.showLogs {
				p = NewPluginManagerOpts()
			} else {
				p = NewPluginManagerOpts(DisableLogs())
			}
			p.ResetLogMode()
			assert.Equal(t, tc.expected, p.GetLogMode())
		})
	}
}

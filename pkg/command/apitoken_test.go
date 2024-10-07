// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/otiai10/copy"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"

	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/auth/uaa"
)

// mockTanzuLogin is a mock implementation of the TanzuLogin function
type mockTanzuLogin struct {
	mock.Mock
}

func (m *mockTanzuLogin) TanzuLogin(issuerURL string, opts ...common.LoginOption) (*common.Token, error) {
	args := m.Called(issuerURL, opts)
	return args.Get(0).(*common.Token), args.Error(1)
}

func TestCreateAPIToken(t *testing.T) {
	var configFile, configFileNG *os.File
	var err error

	setupEnv := func() {
		configFile, err = os.CreateTemp("", "config")
		assert.NoError(t, err)

		err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config.yaml"), configFile.Name())
		assert.NoError(t, err)

		os.Setenv("TANZU_CONFIG", configFile.Name())

		configFileNG, err = os.CreateTemp("", "config_ng")
		assert.NoError(t, err)

		os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())
		err = copy.Copy(filepath.Join("..", "fakes", "config", "tanzu_config_ng.yaml"), configFileNG.Name())
		assert.NoError(t, err)
	}

	teardownEnv := func() {
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")
		os.RemoveAll(configFile.Name())
		os.RemoveAll(configFileNG.Name())
	}

	tests := []struct {
		name          string
		context       *configtypes.Context
		tanzuLoginErr error
		wantErr       bool
		errMsg        string
		output        string
	}{
		{
			name: "success",
			context: &configtypes.Context{
				Name:        "fakecontext",
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Auth: configtypes.GlobalServerAuth{
						Issuer: "https://example.com",
					},
				},
				AdditionalMetadata: map[string]interface{}{
					config.TanzuIdpTypeKey: config.UAAIdpType,
				},
			},
			tanzuLoginErr: nil,
			wantErr:       false,
			output: `==

API Token Generation Successful! Your generated API token is: refresh-token

For Tanzu CLI use in non-interactive settings, set the environment variable TANZU_API_TOKEN=refresh-token before authenticating with the command tanzu login --endpoint <tanzu-platform-endpoint>

Please copy and save your token securely. Note that you will need to regenerate a new token before expiration time and login again to continue using the CLI.
`,
		},
		{
			name: "success with specific endpoint in log message",
			context: &configtypes.Context{
				Name:        "fakecontext",
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Auth: configtypes.GlobalServerAuth{
						Issuer: "https://example.com",
					},
				},
				AdditionalMetadata: map[string]interface{}{
					config.TanzuIdpTypeKey:     config.UAAIdpType,
					config.TanzuHubEndpointKey: "https://platform.tanzu.broadcom.com/hub",
				},
			},
			tanzuLoginErr: nil,
			wantErr:       false,
			output: `==

API Token Generation Successful! Your generated API token is: refresh-token

For Tanzu CLI use in non-interactive settings, set the environment variable TANZU_API_TOKEN=refresh-token before authenticating with the command tanzu login --endpoint https://platform.tanzu.broadcom.com

Please copy and save your token securely. Note that you will need to regenerate a new token before expiration time and login again to continue using the CLI.
`,
		},
		{
			name:          "no active context",
			context:       nil,
			tanzuLoginErr: nil,
			wantErr:       true,
			errMsg:        "no active context found for Tanzu Platform. Please login to Tanzu Platform first to generate an API token",
		},
		{
			name: "invalid active context",
			context: &configtypes.Context{
				Name:        "fakecontext",
				ContextType: configtypes.ContextTypeTanzu,
				AdditionalMetadata: map[string]interface{}{
					config.TanzuIdpTypeKey: config.UAAIdpType,
				},
			},
			tanzuLoginErr: nil,
			wantErr:       true,
			errMsg:        "invalid active context found for Tanzu Platform. Please login to Tanzu Platform first to generate an API token",
		},
		{
			name: "invalid IDP type",
			context: &configtypes.Context{
				Name:        "fakecontext",
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Auth: configtypes.GlobalServerAuth{
						Issuer: "https://example.com",
					},
				},
				AdditionalMetadata: map[string]interface{}{
					config.TanzuIdpTypeKey: "other",
				},
			},
			tanzuLoginErr: nil,
			wantErr:       true,
			errMsg:        "command no supported. Please refer to documentation on how to generate an API token for a public Tanzu Platform endpoint via https://console.tanzu.broadcom.com",
		},
		{
			name: "TanzuLogin error",
			context: &configtypes.Context{
				Name:        "fakecontext",
				ContextType: configtypes.ContextTypeTanzu,
				GlobalOpts: &configtypes.GlobalServer{
					Auth: configtypes.GlobalServerAuth{
						Issuer: "https://example.com",
					},
				},
				AdditionalMetadata: map[string]interface{}{
					config.TanzuIdpTypeKey: config.UAAIdpType,
				},
			},
			tanzuLoginErr: errors.New("TanzuLogin error"),
			wantErr:       true,
			errMsg:        "unable to login, TanzuLogin error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupEnv()
			defer teardownEnv()

			cmd := &cobra.Command{}
			outputBuf := &bytes.Buffer{}
			cmd.SetOutput(outputBuf)

			originalTanzuLogin := uaa.TanzuLogin
			defer func() {
				uaa.TanzuLogin = originalTanzuLogin
			}()

			mockTanzuLogin := &mockTanzuLogin{}
			uaa.TanzuLogin = mockTanzuLogin.TanzuLogin
			if tt.context != nil && tt.context.GlobalOpts != nil && tt.context.AdditionalMetadata[config.TanzuIdpTypeKey] == config.UAAIdpType {
				mockTanzuLogin.On("TanzuLogin", tt.context.GlobalOpts.Auth.Issuer, mock.Anything).
					Return(&common.Token{RefreshToken: "refresh-token", ExpiresIn: 3600}, tt.tanzuLoginErr)
			}

			if tt.context != nil {
				err = config.SetContext(tt.context, true)
				assert.NoError(t, err)
			}

			err := createAPIToken(cmd, nil)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if diff := cmp.Diff(tt.output, outputBuf.String()); diff != "" {
					t.Errorf("Unexpected output (-expected, +actual): %s", diff)
				}
			}

			mockTanzuLogin.AssertExpectations(t)
		})
	}
}

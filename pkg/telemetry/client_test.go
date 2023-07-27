// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/common"
	"github.com/vmware-tanzu/tanzu-cli/pkg/fakes"
	"github.com/vmware-tanzu/tanzu-cli/pkg/plugincmdtree"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
)

func TestClient(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Telemetry plugin test suite")
}

type mockMetricsDB struct {
	createSchemaCalled             bool
	saveOperationMetricCalled      bool
	getRowCountCalled              bool
	createSchemaReturnError        error
	saveOperationMetricReturnError error
	getRowCountError               error
	getRowCountReturnVal           int
}

func (mc *mockMetricsDB) CreateSchema() error {
	mc.createSchemaCalled = true
	return mc.createSchemaReturnError
}

func (mc *mockMetricsDB) SaveOperationMetric(payload *OperationMetricsPayload) error {
	mc.saveOperationMetricCalled = true
	return mc.saveOperationMetricReturnError
}
func (mc *mockMetricsDB) GetRowCount() (int, error) {
	mc.getRowCountCalled = true
	return mc.getRowCountReturnVal, mc.getRowCountError
}

var _ = Describe("Unit tests for UpdateCmdPreRunMetrics()", func() {
	const True = "true"
	var (
		tc           *telemetryClient
		metricsDB    *mockMetricsDB
		rootCmd      *cobra.Command
		cmd          *cobra.Command
		configFile   *os.File
		configFileNG *os.File
		cmdTreeCache *fakes.CommandTreeCache
		err          error
	)

	BeforeEach(func() {
		metricsDB = &mockMetricsDB{}
		cmdTreeCache = &fakes.CommandTreeCache{}
		tc = &telemetryClient{
			currentOperationMetrics: &OperationMetricsPayload{
				StartTime: time.Time{},
			},
			metricsDB: metricsDB,
			cmdTreeCacheGetter: func() (plugincmdtree.Cache, error) {
				return cmdTreeCache, nil
			},
		}

		rootCmd = &cobra.Command{
			Use: "tanzu",
			RunE: func(cmd *cobra.Command, args []string) error {
				return nil
			},
		}

		configFile, err = os.CreateTemp("", "config")
		Expect(err).To(BeNil())
		os.Setenv("TANZU_CONFIG", configFile.Name())

		configFileNG, err = os.CreateTemp("", "config_ng")
		Expect(err).To(BeNil())
		os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())

		err = configlib.SetCLIId("fake-cli-id")
		Expect(err).ToNot(HaveOccurred(), "failed to set the CLI ID")

	})
	AfterEach(func() {
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")

		os.RemoveAll(configFile.Name())
		os.RemoveAll(configFileNG.Name())

	})
	Describe("when the command is CLI core command", func() {
		BeforeEach(func() {
			cmd = &cobra.Command{
				Use: "corecommand",
				RunE: func(cmd *cobra.Command, args []string) error {
					return nil
				},
			}
			rootCmd.AddCommand(cmd)
		})
		Context("when core command has arguments provided but no flags", func() {
			It("should return success and the metrics should have name arg(first argument) hashed and flags string should be empty", func() {
				//tc.UpdateCoreCommandMap(cmd)
				err = tc.UpdateCmdPreRunMetrics(cmd, []string{"arg1"})
				Expect(err).ToNot(HaveOccurred())
				metricsPayload := tc.currentOperationMetrics
				Expect(metricsPayload.CommandName).To(Equal("corecommand"))
				Expect(metricsPayload.NameArg).To(Equal(hashString("arg1")))
				Expect(metricsPayload.Flags).To(BeEmpty())
				Expect(metricsPayload.CliID).ToNot(BeEmpty())
				Expect(metricsPayload.StartTime.IsZero()).To(BeFalse())

			})
		})
		Context("when the core command (requiring hashed values) has arguments and flags provided", func() {
			It("should return success and the metrics should have name arg(first argument) hashed and flags values should be hashed except boolean type flags", func() {
				//tc.UpdateCoreCommandMap(cmd)
				cmd.Flags().String("flag1", "value1", "Flag 1")
				cmd.Flags().Bool("flag2", false, "Flag 2")
				err = cmd.Flags().Set("flag1", "value2")
				Expect(err).ToNot(HaveOccurred())
				err = cmd.Flags().Set("flag2", True)
				Expect(err).ToNot(HaveOccurred())

				err = tc.UpdateCmdPreRunMetrics(cmd, []string{"arg1"})
				Expect(err).ToNot(HaveOccurred())
				metricsPayload := tc.currentOperationMetrics
				Expect(metricsPayload.CommandName).To(Equal("corecommand"))
				Expect(metricsPayload.NameArg).To(Equal(hashString("arg1")))
				flagMap := make(map[string]string)
				flagMap["flag1"] = hashString("value2")
				// boolean flag values should not be hashed
				flagMap["flag2"] = True
				flagStr, err := json.Marshal(flagMap)
				Expect(err).ToNot(HaveOccurred())
				Expect(metricsPayload.Flags).To(Equal(string(flagStr)))
				Expect(metricsPayload.CliID).ToNot(BeEmpty())
				Expect(metricsPayload.StartTime.IsZero()).To(BeFalse())

			})
		})
		Context("when the core command (not requiring hashed values) has arguments and flags provided", func() {
			It("should return success and the metrics should have name arg(first argument) hashed and flags values should be hashed", func() {
				// change the command name to "plugin" since CLI wouldn't hash the flag values for plugin LCM commands
				cmd.Use = "plugin"
				//tc.UpdateCoreCommandMap(cmd)
				cmd.Flags().String("flag1", "value1", "Flag 1")
				cmd.Flags().Bool("flag2", false, "Flag 2")
				err = cmd.Flags().Set("flag1", "value2")
				Expect(err).ToNot(HaveOccurred())
				err = cmd.Flags().Set("flag2", True)
				Expect(err).ToNot(HaveOccurred())

				Expect(configFileNG).ToNot(BeNil())
				err = tc.UpdateCmdPreRunMetrics(cmd, []string{"arg1"})
				Expect(err).ToNot(HaveOccurred())
				metricsPayload := tc.currentOperationMetrics
				Expect(metricsPayload.CommandName).To(Equal("plugin"))
				Expect(metricsPayload.NameArg).To(Equal(hashString("arg1")))
				Expect(metricsPayload.CliID).ToNot(BeEmpty())
				Expect(metricsPayload.StartTime.IsZero()).To(BeFalse())

				flagMap := make(map[string]string)
				flagMap["flag1"] = "value2"
				flagMap["flag2"] = True
				flagStr, err := json.Marshal(flagMap)
				Expect(err).ToNot(HaveOccurred())
				Expect(metricsPayload.Flags).To(Equal(string(flagStr)))

			})
		})
	})

	Describe("when the command is plugin command", func() {

		//Since cobra can only recognize the plugin command in the current plugin architecture, all the subcommands and flags are considered as args for the plugin command
		Context("when plugin command has only args ", func() {
			It("should return success and the metrics should have name arg(first argument) empty and flags string should be empty", func() {
				globalPluginCmd := &cobra.Command{
					Use: "plugin1",
					RunE: func(cmd *cobra.Command, args []string) error {
						return nil
					},
					Annotations: map[string]string{
						"type":                   common.CommandTypePlugin,
						"pluginInstallationPath": "/path/to/plugin1",
					},
				}
				rootCmd.AddCommand(globalPluginCmd)

				tc.SetInstalledPlugins([]cli.PluginInfo{{
					Name:             "plugin1",
					Version:          "1.0.0",
					InstallationPath: "/path/to/plugin1",
				}})

				cmdTreeCache.GetTreeReturns(nil, errors.New("fake-get-command-tree-error"))

				// command: tanzu plugin1 arg1
				err = tc.UpdateCmdPreRunMetrics(globalPluginCmd, []string{"arg1"})

				Expect(err).ToNot(HaveOccurred())
				metricsPayload := tc.currentOperationMetrics
				Expect(metricsPayload.CommandName).To(Equal("plugin1"))
				Expect(metricsPayload.PluginName).To(Equal("plugin1"))
				Expect(metricsPayload.PluginVersion).To(Equal("1.0.0"))
				Expect(metricsPayload.Flags).To(BeEmpty())
				Expect(metricsPayload.CliID).ToNot(BeEmpty())
				Expect(metricsPayload.StartTime.IsZero()).To(BeFalse())

			})
		})

		//Since cobra can only recognize the plugin command in the current plugin architecture, all the subcommands and flags are considered as args for the plugin command
		Context("when kubernetes plugin command has subcommands, args, flags and plugin command tree cache fails to return the plugin command tree ", func() {
			It("should return success and the metrics should have name arg(first argument) empty and flags string should not be empty", func() {

				k8sTargetCmd := &cobra.Command{
					Use: "kubernetes",
					RunE: func(cmd *cobra.Command, args []string) error {
						return nil
					},
				}
				k8sPlugincmd := &cobra.Command{
					Use: "k8s-plugin1",
					RunE: func(cmd *cobra.Command, args []string) error {
						return nil
					},
					Annotations: map[string]string{
						"type":                   common.CommandTypePlugin,
						"pluginInstallationPath": "/path/to/k8s-plugin1",
					},
				}
				k8sTargetCmd.AddCommand(k8sPlugincmd)
				rootCmd.AddCommand(k8sTargetCmd)

				tc.SetInstalledPlugins([]cli.PluginInfo{{
					Name:             "k8s-plugin1",
					Version:          "1.0.0",
					Target:           configtypes.TargetK8s,
					InstallationPath: "/path/to/k8s-plugin1",
				}})

				cmdTreeCache.GetTreeReturns(nil, errors.New("fake-get-command-tree-error"))

				// command : tanzu kubernetes k8s-plugin1 plugin-subcmd1 -v 6 plugin-subcmd2 -ab --flag1=value1 --flag2 value2 -- --arg1 --arg2
				err = tc.UpdateCmdPreRunMetrics(k8sPlugincmd, []string{"plugin-subcmd1", "-v", "6", "plugin-subcmd2", "-ab", "--flag1=value1", "--flag2", "value2", "--", "--arg1", "--arg2"})

				Expect(err).ToNot(HaveOccurred())
				metricsPayload := tc.currentOperationMetrics
				Expect(metricsPayload.CommandName).To(Equal("kubernetes k8s-plugin1"))
				Expect(metricsPayload.PluginName).To(Equal("k8s-plugin1"))
				Expect(metricsPayload.PluginVersion).To(Equal("1.0.0"))
				Expect(metricsPayload.Target).To(Equal(configtypes.TargetK8s))
				Expect(metricsPayload.Endpoint).To(BeEmpty())
				Expect(metricsPayload.NameArg).To(BeEmpty())

				flagMap := make(map[string]string)
				err = json.Unmarshal([]byte(metricsPayload.Flags), &flagMap)
				Expect(err).ToNot(HaveOccurred())

				Expect(flagMap).To(Equal(map[string]string{
					"v":     "",
					"a":     "",
					"b":     "",
					"flag1": "",
					"flag2": "",
				}))
				Expect(metricsPayload.CliID).ToNot(BeEmpty())
				Expect(metricsPayload.StartTime.IsZero()).To(BeFalse())
			})
		})

		//Since cobra can only recognize the plugin command in the current plugin architecture, all the subcommands and flags are considered as args for the plugin command
		Describe("when kubernetes plugin command has subcommands, args, flags and plugin command tree cache returns the command tree ", func() {
			var (
				pluginCMDTree *plugincmdtree.CommandNode
				k8sPlugincmd  *cobra.Command
			)
			BeforeEach(func() {
				k8sTargetCmd := &cobra.Command{
					Use: "kubernetes",
					RunE: func(cmd *cobra.Command, args []string) error {
						return nil
					},
				}
				k8sPlugincmd = &cobra.Command{
					Use: "k8s-plugin1",
					RunE: func(cmd *cobra.Command, args []string) error {
						return nil
					},
					Annotations: map[string]string{
						"type":                   common.CommandTypePlugin,
						"pluginInstallationPath": "/path/to/k8s-plugin1",
					},
				}
				k8sTargetCmd.AddCommand(k8sPlugincmd)
				rootCmd.AddCommand(k8sTargetCmd)
				tc.SetInstalledPlugins([]cli.PluginInfo{{
					Name:             "k8s-plugin1",
					Version:          "1.0.0",
					Target:           configtypes.TargetK8s,
					InstallationPath: "/path/to/k8s-plugin1",
				}})

				// cmd tree for "k8s-plugin1" plugin that would be used by parser for parsing the command args and return command path
				pluginCMDTree = &plugincmdtree.CommandNode{
					Subcommands: map[string]*plugincmdtree.CommandNode{
						"plugin-subcmd1": &plugincmdtree.CommandNode{
							Subcommands: map[string]*plugincmdtree.CommandNode{
								"plugin-subcmd2": plugincmdtree.NewCommandNode(),
							},
							Aliases: map[string]struct{}{
								"pscmd1-alias": {},
							},
						},
					},
				}
			})
			Context("When the user command string matches accurately with plugin command tree ", func() {
				It("should return success and the metrics should have the command path updated", func() {

					cmdTreeCache.GetTreeReturns(pluginCMDTree, nil)

					// command : tanzu kubernetes k8s-plugin1 plugin-subcmd1 -v 6 plugin-subcmd2 -ab --flag1=value1 --flag2 value2 -- --arg1 --arg2
					err = tc.UpdateCmdPreRunMetrics(k8sPlugincmd, []string{"plugin-subcmd1", "-v", "6", "plugin-subcmd2", "-ab", "--flag1=value1", "--flag2", "value2", "--", "--arg1", "--arg2"})

					Expect(err).ToNot(HaveOccurred())
					metricsPayload := tc.currentOperationMetrics
					Expect(metricsPayload.CommandName).To(Equal("kubernetes k8s-plugin1 plugin-subcmd1 plugin-subcmd2"))
					Expect(metricsPayload.PluginName).To(Equal("k8s-plugin1"))
					Expect(metricsPayload.PluginVersion).To(Equal("1.0.0"))
					Expect(metricsPayload.Target).To(Equal(configtypes.TargetK8s))
					Expect(metricsPayload.Endpoint).To(BeEmpty())
					Expect(metricsPayload.NameArg).To(BeEmpty())

					flagMap := make(map[string]string)
					err = json.Unmarshal([]byte(metricsPayload.Flags), &flagMap)
					Expect(err).ToNot(HaveOccurred())

					Expect(flagMap).To(Equal(map[string]string{
						"v":     "",
						"a":     "",
						"b":     "",
						"flag1": "",
						"flag2": "",
					}))
					Expect(metricsPayload.CliID).ToNot(BeEmpty())
					Expect(metricsPayload.StartTime.IsZero()).To(BeFalse())
				})
			})
			Context("When the user command string having command alias matches with plugin command tree ", func() {
				It("should return success and the metrics should have the command path updated correctly", func() {

					cmdTreeCache.GetTreeReturns(pluginCMDTree, nil)

					// command : tanzu kubernetes k8s-plugin1 plugin-subcmd1 -v 6 plugin-subcmd2 -ab --flag1=value1 --flag2 value2 -- --arg1 --arg2
					err = tc.UpdateCmdPreRunMetrics(k8sPlugincmd, []string{"pscmd1-alias", "-v", "6", "plugin-subcmd2", "-ab", "--flag1=value1", "--flag2", "value2", "--", "--arg1", "--arg2"})

					Expect(err).ToNot(HaveOccurred())
					metricsPayload := tc.currentOperationMetrics
					Expect(metricsPayload.CommandName).To(Equal("kubernetes k8s-plugin1 pscmd1-alias plugin-subcmd2"))
					Expect(metricsPayload.PluginName).To(Equal("k8s-plugin1"))
					Expect(metricsPayload.PluginVersion).To(Equal("1.0.0"))
					Expect(metricsPayload.Target).To(Equal(configtypes.TargetK8s))
					Expect(metricsPayload.Endpoint).To(BeEmpty())
					Expect(metricsPayload.NameArg).To(BeEmpty())

					flagMap := make(map[string]string)
					err = json.Unmarshal([]byte(metricsPayload.Flags), &flagMap)
					Expect(err).ToNot(HaveOccurred())

					Expect(flagMap).To(Equal(map[string]string{
						"v":     "",
						"a":     "",
						"b":     "",
						"flag1": "",
						"flag2": "",
					}))
					Expect(metricsPayload.CliID).ToNot(BeEmpty())
					Expect(metricsPayload.StartTime.IsZero()).To(BeFalse())
				})
			})
			Context("When the user command string partially matches with plugin command tree ", func() {
				It("should return success and the metrics should have command path updated upto the point of command match(best-effort)", func() {
					cmdTreeCache.GetTreeReturns(pluginCMDTree, nil)

					// command : tanzu kubernetes k8s-plugin1 plugin-subcmd1 -v 6 plugin-subcmd2 -ab --flag1=value1 --flag2 value2 -- --arg1 --arg2
					err = tc.UpdateCmdPreRunMetrics(k8sPlugincmd, []string{"plugin-subcmd1", "-v", "6", "plugin-subcmd-notmatched", "-ab", "--flag1=value1", "--flag2", "value2", "--", "--arg1", "--arg2"})

					Expect(err).ToNot(HaveOccurred())
					metricsPayload := tc.currentOperationMetrics
					Expect(metricsPayload.CommandName).To(Equal("kubernetes k8s-plugin1 plugin-subcmd1"))
					Expect(metricsPayload.PluginName).To(Equal("k8s-plugin1"))
					Expect(metricsPayload.PluginVersion).To(Equal("1.0.0"))
					Expect(metricsPayload.Target).To(Equal(configtypes.TargetK8s))
					Expect(metricsPayload.Endpoint).To(BeEmpty())
					Expect(metricsPayload.NameArg).To(BeEmpty())

					flagMap := make(map[string]string)
					err = json.Unmarshal([]byte(metricsPayload.Flags), &flagMap)
					Expect(err).ToNot(HaveOccurred())

					Expect(flagMap).To(Equal(map[string]string{
						"v":     "",
						"a":     "",
						"b":     "",
						"flag1": "",
						"flag2": "",
					}))
					Expect(metricsPayload.CliID).ToNot(BeEmpty())
					Expect(metricsPayload.StartTime.IsZero()).To(BeFalse())
				})
			})
		})

	})
})

func TestClient_Client(t *testing.T) {
	client := Client()

	// Call Client() multiple times and ensure that the same instance of the MetricsHandler is returned
	for i := 0; i < 5; i++ {
		assert.Equal(t, client, Client())
	}
}

func TestTelemetryClient_UpdateCmdPostRunMetrics(t *testing.T) {
	tc := &telemetryClient{
		currentOperationMetrics: &OperationMetricsPayload{},
		metricsDB:               &mockMetricsDB{},
	}
	metrics := &PostRunMetrics{
		ExitCode: 1,
	}

	err := tc.UpdateCmdPostRunMetrics(metrics)
	if err != nil {
		t.Errorf("Failed to update post-run metrics: %v", err)
	}

	payload := tc.currentOperationMetrics

	if payload.ExitStatus != 1 {
		t.Errorf("Exit status is not set correctly")
	}

	if payload.EndTime.IsZero() {
		t.Errorf("End time is not set")
	}

	// Test when post run metric data is nil (should not update the metrics)
	if err := tc.UpdateCmdPostRunMetrics(nil); err == nil {
		t.Errorf("Updating command post run metrics should return error if post run metric data is nil")
	}
}

var _ = Describe("Unit tests for SaveMetrics()", func() {
	var (
		tc        *telemetryClient
		metricsDB *mockMetricsDB
		err       error
	)
	BeforeEach(func() {
		metricsDB = &mockMetricsDB{}
		tc = &telemetryClient{
			currentOperationMetrics: &OperationMetricsPayload{
				StartTime: time.Time{},
			},
			metricsDB: metricsDB,
		}
	})

	Context("when the start time is zero", func() {
		It("should return success and not save the metrics to DB", func() {
			err = tc.SaveMetrics()
			Expect(err).ToNot(HaveOccurred())
			Expect(metricsDB.createSchemaCalled).To(BeFalse())
			Expect(metricsDB.saveOperationMetricCalled).To(BeFalse())

		})
	})
	Context("when the DB schema creation failed", func() {
		It("should return failure", func() {
			tc.currentOperationMetrics.StartTime = time.Now()
			metricsDB.createSchemaReturnError = errors.New("fake schema creation error")

			err = tc.SaveMetrics()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("unable to create the telemetry schema: fake schema creation error"))
			Expect(metricsDB.createSchemaCalled).To(BeTrue())
			Expect(metricsDB.saveOperationMetricCalled).To(BeFalse())
		})
	})
	Context("when DB returns error to save the metrics", func() {
		It("should return failure", func() {
			tc.currentOperationMetrics.StartTime = time.Now()
			metricsDB.saveOperationMetricReturnError = errors.New("fake save metrics error")

			err = tc.SaveMetrics()
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(Equal("fake save metrics error"))
			Expect(metricsDB.createSchemaCalled).To(BeTrue())
			Expect(metricsDB.saveOperationMetricCalled).To(BeTrue())
		})
	})
	Context("when metrics is saved in DB successfully", func() {
		It("should return success", func() {
			tc.currentOperationMetrics.StartTime = time.Now()
			err = tc.SaveMetrics()
			Expect(err).ToNot(HaveOccurred())
			Expect(metricsDB.createSchemaCalled).To(BeTrue())
			Expect(metricsDB.saveOperationMetricCalled).To(BeTrue())
		})
	})

})

var _ = Describe("Unit tests for SendMetrics()", func() {
	var (
		tc           *telemetryClient
		metricsDB    *mockMetricsDB
		configFile   *os.File
		configFileNG *os.File
		err          error
	)
	BeforeEach(func() {
		metricsDB = &mockMetricsDB{}
		tc = &telemetryClient{
			currentOperationMetrics: &OperationMetricsPayload{
				StartTime: time.Time{},
			},
			metricsDB: metricsDB,
		}
		configFile, err = os.CreateTemp("", "config")
		Expect(err).To(BeNil())
		os.Setenv("TANZU_CONFIG", configFile.Name())

		configFileNG, err = os.CreateTemp("", "config_ng")
		Expect(err).To(BeNil())
		os.Setenv("TANZU_CONFIG_NEXT_GEN", configFileNG.Name())

		err = configlib.SetCEIPOptIn("true")
		Expect(err).ToNot(HaveOccurred(), "failed to set the CEIP OptIn")

	})
	AfterEach(func() {
		os.Unsetenv("TANZU_CONFIG")
		os.Unsetenv("TANZU_CONFIG_NEXT_GEN")

		os.RemoveAll(configFile.Name())
		os.RemoveAll(configFileNG.Name())

	})

	Context("when the user had opted-out from CEIP", func() {
		It("should return success and should not call DB rouw count", func() {
			err = configlib.SetCEIPOptIn("false")
			Expect(err).ToNot(HaveOccurred(), "failed to set the CEIP OptOut")

			err = tc.SendMetrics(context.Background(), 0)
			Expect(err).ToNot(HaveOccurred())
			Expect(metricsDB.getRowCountCalled).To(BeFalse())
		})
	})
	Context("when the user had opted-in for CEIP and DB row count is less than send rowcount threshold", func() {
		It("should return success and not call send operation", func() {
			metricsDB.getRowCountReturnVal = metricsSendThresholdRowCount - 1

			err = tc.SendMetrics(context.Background(), 1)
			Expect(err).ToNot(HaveOccurred())
			Expect(metricsDB.getRowCountCalled).To(BeTrue())
		})
	})
	Context("when the send metrics conditions are met, but the telemetry plugin was not installed", func() {
		It("should return error", func() {
			metricsDB.getRowCountReturnVal = metricsSendThresholdRowCount
			tc.SetInstalledPlugins(nil)
			err = tc.SendMetrics(context.Background(), 1)
			Expect(err).To(HaveOccurred())
			Expect(metricsDB.getRowCountCalled).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring("unable to get the telemetry plugin"))

		})
	})

	Context("when the send metrics conditions are met, but the telemetry plugin installation path was incorrect", func() {
		It("should return error", func() {
			metricsDB.getRowCountReturnVal = metricsSendThresholdRowCount
			tc.SetInstalledPlugins([]cli.PluginInfo{{
				Name:             "telemetry",
				Target:           configtypes.TargetGlobal,
				InstallationPath: "incorrect/path",
				Version:          "1.0.0",
			}})
			err = tc.SendMetrics(context.Background(), 1)
			Expect(err).To(HaveOccurred())
			Expect(metricsDB.getRowCountCalled).To(BeTrue())
			Expect(err.Error()).To(ContainSubstring(`plugin "telemetry" does not exist`))
		})
	})

})

func TestTelemetryClient_isCoreCommand(t *testing.T) {
	coreCMD := &cobra.Command{
		Use: "core-command-wo-annotations",
		RunE: func(cmd *cobra.Command, args []string) error {
			return nil
		},
	}
	isCoreCMD := isCoreCommand(coreCMD)
	if !isCoreCMD {
		t.Error("isCoreCommand should return true for a command with out annotations")
	}

	// set the annotation, but without 'type': 'plugin'
	coreCMD.Annotations = map[string]string{
		"group": "System",
	}
	isCoreCMD = isCoreCommand(coreCMD)
	if !isCoreCMD {
		t.Error("isCoreCommand should return true for the command with out plugin annotation")
	}

	// set the annotation, but without 'type': 'plugin'
	coreCMD.Annotations = map[string]string{
		"type": common.CommandTypePlugin,
	}
	isCoreCMD = isCoreCommand(coreCMD)
	if isCoreCMD {
		t.Error("isCoreCommand should return false for the command with plugin annotation")
	}
}

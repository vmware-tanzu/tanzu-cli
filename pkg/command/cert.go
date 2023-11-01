// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package command

import (
	"encoding/base64"
	"fmt"
	"os"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"

	"github.com/vmware-tanzu/tanzu-cli/pkg/cli"
	"github.com/vmware-tanzu/tanzu-cli/pkg/utils"
)

var (
	host, caCertPathForAdd, skipCertVerifyForAdd, insecureForAdd    string
	caCertPathForUpdate, skipCertVerifyForUpdate, insecureForUpdate string
)

func newCertCmd() *cobra.Command {
	var certCmd = &cobra.Command{
		Use:   "cert",
		Short: "Manage certificate configuration of hosts",
		Long:  "Manage certificate configuration of hosts",
	}
	certCmd.SetUsageFunc(cli.SubCmdUsageFunc)

	listCertCmd := newListCertCmd()
	addCertCmd := newAddCertCmd()
	updateCertCmd := newUpdateCertCmd()
	deleteCertCmd := newDeleteCertCmd()

	compSkipFlag := func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{
				"true\tSkip TLS certificate verification (insecure)",
				"false\tPerform TLS certificate verification"},
			cobra.ShellCompDirectiveNoFileComp
	}

	compInsecureFlag := func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return []string{
				"true\tAllow the use of http when interacting with the host (insecure)",
				"false\tPrevent the use of http when interacting with the host"},
			cobra.ShellCompDirectiveNoFileComp
	}

	addCertCmd.Flags().StringVarP(&host, "host", "", "", "host or host:port")
	utils.PanicOnErr(addCertCmd.RegisterFlagCompletionFunc("host", func(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
		return cobra.AppendActiveHelp(nil, "Please provide 'host' or 'host:port'"), cobra.ShellCompDirectiveNoFileComp
	}))

	// --ca-certificate is renamed to --ca-cert
	addCertCmd.Flags().StringVarP(&caCertPathForAdd, "ca-certificate", "", "", "path to the public certificate")
	caCertificateDeprecationMsg := "this was done in the v1.1.0 release, it will be removed following the deprecation policy (6 months). Use the --ca-cert flag instead.\n"
	utils.PanicOnErr(addCertCmd.Flags().MarkDeprecated("ca-certificate", caCertificateDeprecationMsg))
	// The completion for this flag is simple file completion, which is configured by default
	addCertCmd.Flags().StringVarP(&caCertPathForAdd, "ca-cert", "", "", "path to the public certificate")

	addCertCmd.Flags().StringVarP(&skipCertVerifyForAdd, "skip-cert-verify", "", "false", "skip server's TLS certificate verification")
	utils.PanicOnErr(addCertCmd.RegisterFlagCompletionFunc("skip-cert-verify", compSkipFlag))

	addCertCmd.Flags().StringVarP(&insecureForAdd, "insecure", "", "false", "allow the use of http when interacting with the host")
	utils.PanicOnErr(addCertCmd.RegisterFlagCompletionFunc("insecure", compInsecureFlag))

	utils.PanicOnErr(cobra.MarkFlagRequired(addCertCmd.Flags(), "host"))

	// --ca-certificate is renamed to --ca-cert
	updateCertCmd.Flags().StringVarP(&caCertPathForUpdate, "ca-certificate", "", "", "path to the public certificate")
	utils.PanicOnErr(updateCertCmd.Flags().MarkDeprecated("ca-certificate", caCertificateDeprecationMsg))
	// The completion for this flag is simple file completion, which is configured by default
	updateCertCmd.Flags().StringVarP(&caCertPathForUpdate, "ca-cert", "", "", "path to the public certificate")

	updateCertCmd.Flags().StringVarP(&skipCertVerifyForUpdate, "skip-cert-verify", "", "", "skip server's TLS certificate verification (true|false)")
	utils.PanicOnErr(updateCertCmd.RegisterFlagCompletionFunc("skip-cert-verify", compSkipFlag))

	updateCertCmd.Flags().StringVarP(&insecureForUpdate, "insecure", "", "", "allow the use of http when interacting with the host (true|false)")
	utils.PanicOnErr(updateCertCmd.RegisterFlagCompletionFunc("insecure", compInsecureFlag))

	listCertCmd.Flags().StringVarP(&outputFormat, "output", "o", "", "output format (yaml|json|table)")
	utils.PanicOnErr(listCertCmd.RegisterFlagCompletionFunc("output", completionGetOutputFormats))

	certCmd.AddCommand(
		listCertCmd,
		addCertCmd,
		updateCertCmd,
		deleteCertCmd,
	)

	return certCmd
}

func newListCertCmd() *cobra.Command {
	var listCertsCmd = &cobra.Command{
		Use:               "list",
		Short:             "List available certificate configurations",
		ValidArgsFunction: noMoreCompletions,
		RunE: func(cmd *cobra.Command, args []string) error {
			output := component.NewOutputWriterWithOptions(cmd.OutOrStdout(), outputFormat, []component.OutputWriterOption{}, "host", "ca-certificate", "skip-cert-verification", "insecure")
			certs, _ := configlib.GetCerts()
			for _, cert := range certs {
				// TODO(prkalle): Remove the column "CACertData" if "<REDACTED>" string is not good UX, also would have to change if "Not configured" is not the apt word
				notConfiguredStr := "Not configured"
				caData := notConfiguredStr
				if cert.CACertData != "" {
					caData = "<REDACTED>"
				}
				if cert.SkipCertVerify == "" {
					cert.SkipCertVerify = notConfiguredStr
				}
				if cert.Insecure == "" {
					cert.Insecure = notConfiguredStr
				}
				output.AddRow(cert.Host, caData, cert.SkipCertVerify, cert.Insecure)
			}
			output.Render()
			return nil
		},
	}
	return listCertsCmd
}

func newAddCertCmd() *cobra.Command {
	var addCertCmd = &cobra.Command{
		Use:   "add",
		Short: "Add a certificate configuration for a host",
		Long:  "Add a certificate configuration for a host",
		Example: `
    # Add CA certificate for a host
    tanzu config cert add --host test.vmware.com --ca-cert path/to/ca/ert

    # Add CA certificate for a host:port
    tanzu config cert add --host test.vmware.com:8443 --ca-cert path/to/ca/ert

    # Set to skip verifying the certificate while interacting with host
    tanzu config cert add --host test.vmware.com  --skip-cert-verify true

    # Set to allow insecure (http) connection while interacting with host
    tanzu config cert add --host test.vmware.com  --insecure true`,
		ValidArgsFunction: completeAddCert,
		RunE: func(cmd *cobra.Command, args []string) error {
			if skipCertVerifyForAdd != "" {
				if !strings.EqualFold(skipCertVerifyForAdd, "true") && !strings.EqualFold(skipCertVerifyForAdd, "false") {
					return errors.Errorf("incorrect boolean argument for '--skip-cert-verify' option : %q", skipCertVerifyForAdd)
				}
			}
			if insecureForAdd != "" {
				if !strings.EqualFold(insecureForAdd, "true") && !strings.EqualFold(insecureForAdd, "false") {
					return errors.Errorf("incorrect boolean argument for '--insecure' option : %q", insecureForAdd)
				}
			}
			if strings.EqualFold(skipCertVerifyForAdd, "false") && strings.EqualFold(insecureForAdd, "false") &&
				caCertPathForAdd == "" {
				return errors.New("please specify at least one additional valid option apart from '--host' ")
			}
			certExistError := fmt.Errorf("certificate configuration for host %q already exist", host)
			exits, _ := configlib.CertExists(host)
			if exits {
				return certExistError
			}
			newCert, err := createCert(host, caCertPathForAdd, skipCertVerifyForAdd, insecureForAdd)
			if err != nil {
				return err
			}

			err = configlib.SetCert(newCert)
			if err != nil {
				return err
			}

			log.Successf("successfully added certificate data for host %s", host)
			return nil
		},
	}
	return addCertCmd
}

func newUpdateCertCmd() *cobra.Command {
	var updateCertCmd = &cobra.Command{
		Use:   "update HOST",
		Short: "Update certificate configuration for a host",
		Args:  cobra.ExactArgs(1),
		Example: `
    # Update CA certificate for a host,
    tanzu config cert update test.vmware.com --ca-cert path/to/ca/ert

    # Update CA certificate for a host:port,
    tanzu config cert update test.vmware.com:5443 --ca-cert path/to/ca/ert

    # Update whether to skip verifying the certificate while interacting with host
    tanzu config cert update test.vmware.com  --skip-cert-verify true

    # Update whether to allow insecure (http) connection while interacting with host
    tanzu config cert update test.vmware.com  --insecure true`,
		ValidArgsFunction: completeCertHosts,
		RunE: func(cmd *cobra.Command, args []string) error {
			if skipCertVerifyForUpdate != "" {
				if !strings.EqualFold(skipCertVerifyForUpdate, "true") && !strings.EqualFold(skipCertVerifyForUpdate, "false") {
					return errors.Errorf("incorrect boolean argument for '--skip-cert-verify' option : %q", skipCertVerifyForUpdate)
				}
			}
			if insecureForUpdate != "" {
				if !strings.EqualFold(insecureForUpdate, "true") && !strings.EqualFold(insecureForUpdate, "false") {
					return errors.Errorf("incorrect boolean argument for '--insecure' option : %q", insecureForUpdate)
				}
			}
			if skipCertVerifyForUpdate == "" && insecureForUpdate == "" && caCertPathForUpdate == "" {
				return errors.New("please specify at least one update option ")
			}
			aHost := args[0]
			certNoExistError := fmt.Errorf("certificate configuration for host %q does not exist", aHost)
			exits, _ := configlib.CertExists(aHost)
			if !exits {
				return certNoExistError
			}
			cert, err := createCert(aHost, caCertPathForUpdate, skipCertVerifyForUpdate, insecureForUpdate)
			if err != nil {
				return err
			}

			err = configlib.SetCert(cert)
			if err != nil {
				return err
			}
			log.Successf("updated certificate data for host %s", aHost)
			return nil
		},
	}
	return updateCertCmd
}

func newDeleteCertCmd() *cobra.Command {
	var deleteCertCmd = &cobra.Command{
		Use:   "delete HOST",
		Short: "Delete certificate configuration for a host",
		Args:  cobra.ExactArgs(1),
		Example: `
    # Delete a certificate for host
    tanzu config cert delete test.vmware.com

    # Delete a certificate for host:port
    tanzu config cert delete test.vmware.com:5443`,
		ValidArgsFunction: completeCertHosts,
		RunE: func(cmd *cobra.Command, args []string) (err error) {
			aHost := args[0]

			err = configlib.DeleteCert(aHost)
			if err != nil {
				return err
			}
			log.Successf("deleted certificate data for host %s", aHost)
			return nil
		},
	}
	return deleteCertCmd
}

func createCert(host, caCertPath, skipCertVerify, insecure string) (*configtypes.Cert, error) {
	cert := &configtypes.Cert{
		Host: host,
	}

	if caCertPath != "" {
		fileBytes, err := os.ReadFile(caCertPath)
		if err != nil {
			return nil, errors.Wrapf(err, "error reading CA certificate file %s", caCertPath)
		}
		cert.CACertData = base64.StdEncoding.EncodeToString(fileBytes)
	}

	if skipCertVerify != "" {
		cert.SkipCertVerify = skipCertVerify
	}
	if insecure != "" {
		cert.Insecure = insecure
	}

	return cert, nil
}

// ====================================
// Shell completion functions
// ====================================
func completeCertHosts(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
	}

	var comps []string
	certs, _ := configlib.GetCerts()
	for _, cert := range certs {
		desc := fmt.Sprintf("Insecure: %s, Skip cert verification: %s", cert.Insecure, cert.SkipCertVerify)
		comps = append(comps, fmt.Sprintf("%s\t%s", cert.Host, desc))
	}

	// Sort to allow for testing
	sort.Strings(comps)

	return comps, cobra.ShellCompDirectiveNoFileComp
}

func completeAddCert(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	if host == "" {
		// This flag is required, so completion will be provided for it
		return nil, cobra.ShellCompDirectiveNoFileComp
	}

	// The user has provided enough information
	return activeHelpNoMoreArgs(nil), cobra.ShellCompDirectiveNoFileComp
}

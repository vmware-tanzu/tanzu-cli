// Copyright 2022-23 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registry

import (
	"context"
	"encoding/base64"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	regname "github.com/google/go-containerregistry/pkg/name"
	gocontainerregistry "github.com/google/go-containerregistry/pkg/registry"
	"github.com/pkg/errors"

	"github.com/vmware-tanzu/tanzu-cli/pkg/configpaths"
	"github.com/vmware-tanzu/tanzu-cli/pkg/constants"
	configlib "github.com/vmware-tanzu/tanzu-plugin-runtime/config"
	configtypes "github.com/vmware-tanzu/tanzu-plugin-runtime/config/types"
	tprlog "github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

type CertOptions struct {
	CACertPaths    []string
	SkipCertVerify bool
	Insecure       bool
}

func GetRegistryCertOptions(registryHost string) (*CertOptions, error) {
	registryCertOpts := &CertOptions{
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
		err := checkForProxyConfigAndUpdateCert(registryCertOpts)
		if err != nil {
			return nil, errors.Wrap(err, "failed to check for proxy config and update the cert")
		}
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

	err = checkForProxyConfigAndUpdateCert(registryCertOpts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to check for proxy config and update the cert")
	}
	return registryCertOpts, nil
}

// updateRegistryCertOptions sets the registry options by taking the custom certificate data configured for registry as input
func updateRegistryCertOptions(cert *configtypes.Cert, registryCertOpts *CertOptions) error {
	if cert.SkipCertVerify != "" {
		skipVerifyCerts, _ := strconv.ParseBool(cert.SkipCertVerify)
		registryCertOpts.SkipCertVerify = skipVerifyCerts
	}
	if cert.Insecure != "" {
		insecure, _ := strconv.ParseBool(cert.Insecure)
		registryCertOpts.Insecure = insecure
	}

	err := updateCACertData(cert.CACertData, registryCertOpts)
	if err != nil {
		return err
	}

	return nil
}

// AddRegistryTrustedRootCertsFileForWindows adds CA certificate to registry options for Windows environments
func AddRegistryTrustedRootCertsFileForWindows(registryCertOpts *CertOptions) error {
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
// It also supports a digest format:
// (e.g. localhost:9876/tanzu-cli/plugins/plugin@sha256:3925a7a0e78ec439529c4bc9e26b4bbe95a01645325a8b2f66334be7e6b37ab6)
func GetRegistryName(imageName string) (string, error) {
	ref, err := regname.ParseReference(imageName)
	if err != nil {
		return "", errors.Wrapf(err, "unable to fetch registry name from image %q", imageName)
	}
	return ref.Context().RegistryStr(), nil
}

// checkForProxyConfigAndUpdateCert checks if user has configured proxy CA cert data using "PROXY_CA_CERT" environment variable
// if configured, updates cert data in CertOptions
func checkForProxyConfigAndUpdateCert(registryCertOpts *CertOptions) error {
	// check if user provided cert configuration for proxy host, if so, use it
	proxyCACertData := os.Getenv(constants.ProxyCACert)

	// If proxy CA cert data is available, overwrite the registry cert data
	err := updateCACertData(proxyCACertData, registryCertOpts)
	if err != nil {
		return err
	}

	return nil
}

func updateCACertData(caCertData string, registryCertOpts *CertOptions) error {
	if caCertData != "" {
		caCertBytes, err := base64.StdEncoding.DecodeString(caCertData)
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

// ServeLocalRegistry start an in-memory localhost registry server at provided port
// If the port is not provided it uses a random available port to start the registry server
// It returns the port on which the server is running, the function to shutdown the server, error if any
func ServeLocalRegistry(port string) (string, func(), error) {
	ctx := context.Background()
	listener, err := net.Listen("tcp", "localhost:"+port)
	if err != nil {
		return port, nil, err
	}
	porti := listener.Addr().(*net.TCPAddr).Port
	port = fmt.Sprintf("%d", porti)

	serverLogFile, err := os.CreateTemp("", "")
	if err != nil {
		return port, nil, err
	}
	s := &http.Server{
		ReadHeaderTimeout: 5 * time.Second, // prevent slowloris, quiet linter
		Handler:           gocontainerregistry.New(gocontainerregistry.Logger(log.New(serverLogFile, "", log.LstdFlags))),
		ErrorLog:          nil,
	}
	tprlog.Infof("starting local registry server on port %s, logs available at %s", port, serverLogFile.Name())

	errCh := make(chan error)
	go func() { errCh <- s.Serve(listener) }()

	return port, func() {
		tprlog.Info("shutting down local registry server...")
		if err := s.Shutdown(ctx); err != nil {
			tprlog.Infof("error while terminating local registry server: %v", err.Error())
		}
		if err := <-errCh; !errors.Is(err, http.ErrServerClosed) {
			tprlog.Infof("error received from local registry server: %v", err.Error())
		}
		os.Remove(serverLogFile.Name())
	}, nil
}

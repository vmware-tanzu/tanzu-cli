// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package cosignhelper implements cosign verification functionality using cosign libraries
package cosignhelper

import (
	"context"
	"crypto"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net/http"
	"os"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/google/go-containerregistry/pkg/v1/remote"
	"github.com/pkg/errors"
	"github.com/sigstore/cosign/v2/pkg/cosign"
	"github.com/sigstore/cosign/v2/pkg/cosign/pkcs11key"
	ociremote "github.com/sigstore/cosign/v2/pkg/oci/remote"
	sigs "github.com/sigstore/cosign/v2/pkg/signature"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
	kerrors "k8s.io/apimachinery/pkg/util/errors"
)

// RegistryOptions registry options used while interacting with registry
type RegistryOptions struct {
	// CACertPaths is the path to CA certs for the registry endpoint.
	// This would be required if the registry is self-signed
	CACertPaths []string
	// SkipCertVerify is to allow insecure connections to registries (e.g., with expired or self-signed TLS certificates)
	SkipCertVerify bool
	// AllowInsecure is to allow using HTTP instead of HTTPS protocol while connecting to registries
	AllowInsecure bool
}

// CosignVerifyOptions implements the "cosign verify" command using cosign library
type CosignVerifyOptions struct {
	// PublicKeyPath is the path to custom public key to be used to verify the signature
	// of the OCI image. If the path is empty, the CLI embedded public key would be used
	PublicKeyPath string
	// RegistryOpts registry options used while interacting with registry
	RegistryOpts *RegistryOptions
}

func NewCosignVerifier(publicKeyPath string, registryOpts *RegistryOptions) Cosignhelper {
	return &CosignVerifyOptions{
		PublicKeyPath: publicKeyPath,
		RegistryOpts:  registryOpts,
	}
}

// Verify verifies the signature on the images
func (vo *CosignVerifyOptions) Verify(ctx context.Context, images []string) error {
	var pubKeys []signature.Verifier
	var err error
	httpTrans, err := vo.newHTTPTransport()
	if err != nil {
		return errors.Wrapf(err, "creating registry HTTP transport")
	}
	// TODO: Investigate If CLI need transparency log verification, and add support for RekorURL
	// The Rekor Transparency log verification was experimental in v1.13.1 and regular feature in v2.x.x
	// Using Rekor Default URL and Rekor public Keys (downloaded from online by default) not be feasible for air-gapped environment
	ignoreTlog := true

	switch {
	// If PublicKeyPath is provided(custom public key) use it, else use the embedded public key
	case vo.PublicKeyPath != "":
		pubKey, err := sigs.PublicKeyFromKeyRefWithHashAlgo(ctx, vo.PublicKeyPath, crypto.SHA256)
		if err != nil {
			return fmt.Errorf("loading custom public key: %w", err)
		}
		pubKeys = append(pubKeys, pubKey)
		pkcs11Key, ok := pubKey.(*pkcs11key.Key)
		if ok {
			defer pkcs11Key.Close()
		}

	default:
		for _, raw := range [][]byte{tanzuCLIPluginDBImageSignPublicKeyOfficial} {
			// PEM encoded file.
			key, err := cryptoutils.UnmarshalPEMToPublicKey(raw)
			if err != nil {
				return fmt.Errorf("failed unmarshalling PEM encoded default public key: %w", err)
			}
			pubKey, err := signature.LoadVerifier(key, crypto.SHA256)
			if err != nil {
				return fmt.Errorf("loading default public key: %w", err)
			}
			pubKeys = append(pubKeys, pubKey)
		}
	}

	var nameOpts []name.Option
	if vo.RegistryOpts.AllowInsecure {
		nameOpts = append(nameOpts, name.Insecure)
	}

	for _, img := range images {
		ref, err := name.ParseReference(img, nameOpts...)
		if err != nil {
			return fmt.Errorf("parsing reference: %w", err)
		}

		var arrErr []error
		for _, verifier := range pubKeys {
			co := &cosign.CheckOpts{
				RegistryClientOpts: []ociremote.Option{
					ociremote.WithRemoteOptions(remote.WithContext(ctx)),
					ociremote.WithRemoteOptions(remote.WithTransport(httpTrans)),
				},
				IgnoreTlog:  ignoreTlog,
				SigVerifier: verifier,
			}

			_, _, err = cosign.VerifyImageSignatures(ctx, ref, co)
			if err == nil {
				break // if signature verification successful break the loop
			}
			arrErr = append(arrErr, fmt.Errorf("failed validating the signature of the image %s :%w", img, err))
		}
		// If all the verifier has returned error then mark the verification as failed
		// and return the error
		if len(arrErr) == len(pubKeys) {
			return kerrors.NewAggregate(arrErr)
		}
	}

	return nil
}

func (vo *CosignVerifyOptions) newHTTPTransport() (*http.Transport, error) {
	var pool *x509.CertPool

	var err error
	pool, err = x509.SystemCertPool()
	if err != nil {
		return nil, err
	}

	if len(vo.RegistryOpts.CACertPaths) > 0 {
		for _, path := range vo.RegistryOpts.CACertPaths {
			if certs, err := os.ReadFile(path); err != nil {
				return nil, errors.Wrapf(err, "failed reading CA certificates from '%s' ", path)
			} else if ok := pool.AppendCertsFromPEM(certs); !ok {
				return nil, fmt.Errorf("failed adding CA certificates from '%s'", path)
			}
		}
	}

	clonedDefaultTransport := http.DefaultTransport.(*http.Transport).Clone()
	clonedDefaultTransport.ForceAttemptHTTP2 = false
	// #nosec G402
	clonedDefaultTransport.TLSClientConfig = &tls.Config{
		RootCAs:            pool,
		InsecureSkipVerify: vo.RegistryOpts.SkipCertVerify,
	}

	return clonedDefaultTransport, nil
}

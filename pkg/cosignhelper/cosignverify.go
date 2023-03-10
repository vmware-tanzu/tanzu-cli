// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package cosignhelper implements cosign verification functionality using cosign libraries
package cosignhelper

import (
	"context"
	"crypto"
	"fmt"

	"github.com/google/go-containerregistry/pkg/name"
	"github.com/sigstore/cosign/pkg/cosign"
	"github.com/sigstore/cosign/pkg/cosign/pkcs11key"
	sigs "github.com/sigstore/cosign/pkg/signature"
	"github.com/sigstore/sigstore/pkg/cryptoutils"
	"github.com/sigstore/sigstore/pkg/signature"
)

// CosignVerifyOptions implements the "cosign verify" command using cosign library
type CosignVerifyOptions struct {
	PublicKeyPath string
}

func NewCosignVerifier(publicKeyPath string) Cosignhelper {
	return &CosignVerifyOptions{
		PublicKeyPath: publicKeyPath,
	}
}

// Verify verifies the signature on the images
func (vo *CosignVerifyOptions) Verify(ctx context.Context, images []string) error {
	var pubKey signature.Verifier
	var err error

	co := &cosign.CheckOpts{}

	switch {
	// If PublicKeyPath is provided(custom public key) use it, else use the embedded public key
	case vo.PublicKeyPath != "":
		pubKey, err = sigs.PublicKeyFromKeyRefWithHashAlgo(ctx, vo.PublicKeyPath, crypto.SHA256)
		if err != nil {
			return fmt.Errorf("loading custom public key: %w", err)
		}
		pkcs11Key, ok := pubKey.(*pkcs11key.Key)
		if ok {
			defer pkcs11Key.Close()
		}

	default:
		// use the default embedded cert
		raw := tanzuCLIPluginDBImageSignPublicKey
		// PEM encoded file.
		key, err := cryptoutils.UnmarshalPEMToPublicKey(raw)
		if err != nil {
			return fmt.Errorf("failed unmarshalling PEM encoded default public key: %w", err)
		}
		pubKey, err = signature.LoadVerifier(key, crypto.SHA256)
		if err != nil {
			return fmt.Errorf("loading default public key: %w", err)
		}
	}
	co.SigVerifier = pubKey

	for _, img := range images {
		ref, err := name.ParseReference(img, []name.Option{}...)
		if err != nil {
			return fmt.Errorf("parsing reference: %w", err)
		}
		_, _, err = cosign.VerifyImageSignatures(ctx, ref, co)
		if err != nil {
			return fmt.Errorf("failed validating the signature of the image %s :%w", img, err)
		}
	}
	return nil
}

// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cosignhelper

import "context"

//go:generate counterfeiter -o ../fakes/cosignhelper_fake.go --fake-name Cosignhelperfake . Cosignhelper

// Cosignhelper is the interface to provide wrapper implementation for cosign libraries
type Cosignhelper interface {
	// Verify verifies the signature on the images using cosign library
	Verify(ctx context.Context, images []string) error
}

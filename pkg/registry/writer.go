// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package registry

import "github.com/vmware-tanzu/tanzu-plugin-runtime/log"

type writer struct{}

// Writer passes the log received to the log.Info
func (w *writer) Write(p []byte) (n int, err error) {
	log.Info(string(p))
	return len(p), nil
}

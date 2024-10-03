// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package time contains time specific functions and variable to make it to overwrite for unit tests
package time

import "time"

var Now = time.Now

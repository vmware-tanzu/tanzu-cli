// Copyright 2024 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package globalinit is used to execute different initializers of the CLI
// based on their specified triggers.  This can be used to cleanup some data
// or perform some actions based on certain conditions.
package globalinit

import (
	"io"

	kerrors "k8s.io/apimachinery/pkg/util/errors"
)

type initializer struct {
	// The trigger functions are kept separate from the initialization functions
	// for a couple of reasons:
	// 1. The trigger functions can be used first to determine if the initialization
	//    should be performed.  If so, a global message can be printed to the user
	//    before the initialization is performed.
	// 2. Some trigger functions or initialization functions could be re-used by
	//    different features for slightly different purposes.
	triggerFunc        func() bool
	initializationFunc func(outStream io.Writer) error
}

var (
	initializers []initializer
)

// RegisterInitializer registers a new initializer with the global list of initializers.
// The trigger function is used to determine if the initialization function should be run.
// The initialization function is the function that will be run if the trigger function returns true.
// The set of initializer triggers is checked whenever the CLI is run.
func RegisterInitializer(trigger func() bool, initialization func(writer io.Writer) error) {
	initializers = append(initializers, initializer{triggerFunc: trigger, initializationFunc: initialization})
}

// InitializationRequired checks if any of the registered initializers should be triggered.
func InitializationRequired() bool {
	for _, i := range initializers {
		if i.triggerFunc() {
			return true
		}
	}
	return false
}

// PerformInitializations run each initializer which which the trigger function returns true.
func PerformInitializations(outStream io.Writer) error {
	var errorList []error
	for _, i := range initializers {
		if i.triggerFunc() {
			if err := i.initializationFunc(outStream); err != nil {
				errorList = append(errorList, err)
			}
		}
	}

	return kerrors.NewAggregate(errorList)
}

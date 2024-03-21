// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"os"
	"os/signal"
	"syscall"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/component"
	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

// SignalCatcherInitialization initializes a signal catcher to catch OS signals and stop the spinner
func SignalCatcherInitialization(signalChannel chan os.Signal, s component.OutputWriterSpinner, spinnerFinalText string, spinnerFinalTextLogType log.LogType, errorMsgAfterSpinnerStop string) {
	// Register the channel to receive interrupt signals (e.g., Ctrl+C)
	signal.Notify(signalChannel, syscall.SIGINT, syscall.SIGTERM)

	sig := <-signalChannel
	if sig != nil {
		if s != nil {
			s.SetFinalText(spinnerFinalText, spinnerFinalTextLogType)
			s.StopSpinner()
		}
		log.Errorf(errorMsgAfterSpinnerStop)
		os.Exit(128 + int(sig.(syscall.Signal)))
	}
}

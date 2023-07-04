// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"reflect"
	"testing"
)

func TestTraverseFlagNames(t *testing.T) {
	args := []string{"-a", "--flag1", "--flag2=value", "-bc", "--", "--arg1", "--arg2"}

	expectedFlags := []string{"a", "flag1", "flag2", "b", "c"}
	resultFlags := TraverseFlagNames(args)

	if !reflect.DeepEqual(resultFlags, expectedFlags) {
		t.Errorf("Expected flags: %v, but got: %v", expectedFlags, resultFlags)
	}

	// ex: tanzu pluginName -v 6 pluginCmd --flag1 pluginSubcmd --flag1=10 -bc -- arg1 --arg2
	argsWithPluginCommands := []string{"-v", "6", "pluginCmd", "--flag1", "pluginSubCmd", "--flag2=value", "-bc", "--", "--arg1", "--arg2"}

	expectedFlagsWithoutPluginCommands := []string{"v", "flag1", "flag2", "b", "c"}
	resultFlags = TraverseFlagNames(argsWithPluginCommands)

	if !reflect.DeepEqual(resultFlags, expectedFlagsWithoutPluginCommands) {
		t.Errorf("Expected flags excluding the pluginCommand in the args: %v, but got: %v", expectedFlags, resultFlags)
	}
}

func TestFlagNamesToJSONString(t *testing.T) {
	flagNames := []string{"flag1", "flag2", "flag3"}

	expectedJSONString := `{"flag1":"","flag2":"","flag3":""}`
	resultJSONString := flagNamesToJSONString(flagNames)

	if resultJSONString != expectedJSONString {
		t.Errorf("Expected JSON string: %s, but got: %s", expectedJSONString, resultJSONString)
	}
}

func TestProcessFlagNames(t *testing.T) {
	flags := []string{"a", "flag1", "flag2", "b", "c"}

	expectedResultFlags := []string{"a", "flag1", "flag2", "b", "c"}
	resultFlags := processFlagNames(flags)

	if !reflect.DeepEqual(resultFlags, expectedResultFlags) {
		t.Errorf("Expected flags: %v, but got: %v", expectedResultFlags, resultFlags)
	}
}

func TestIsFlagArg(t *testing.T) {
	flagArg1 := "--flag"
	flagArg2 := "-a"
	nonFlagArg := "arg"

	if !isFlagArg(flagArg1) {
		t.Errorf("Expected %s to be a flag argument", flagArg1)
	}

	if !isFlagArg(flagArg2) {
		t.Errorf("Expected %s to be a flag argument", flagArg2)
	}

	if isFlagArg(nonFlagArg) {
		t.Errorf("Expected %s not to be a flag argument", nonFlagArg)
	}
}

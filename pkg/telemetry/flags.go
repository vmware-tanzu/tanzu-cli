// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package telemetry

import (
	"encoding/json"
	"strings"
)

const doubleHyphen = "--"

func TraverseFlagNames(args []string) []string {
	flags := []string{}
Loop:
	for _, arg := range args {
		switch {
		// "--" terminates the flags (everything after is an argument)
		case arg == doubleHyphen:
			break Loop
		// A long flag with a space separated value
		case strings.HasPrefix(arg, doubleHyphen) && !strings.Contains(arg, "="):
			flags = append(flags, arg)
			continue
		// A short flag with a space separated value
		case strings.HasPrefix(arg, "-") && !strings.Contains(arg, "=") && len(arg) == 2:
			flags = append(flags, arg)
			continue
		// A flag without a value, or with an `=` separated value
		case isFlagArg(arg):
			if strings.Contains(arg, "=") {
				flags = append(flags, strings.Split(arg, "=")[0])
			} else {
				flags = append(flags, arg)
			}
			continue
		}
	}
	return processFlagNames(flags)
}

func flagNamesToJSONString(flagNames []string) string {
	flagMap := make(map[string]string)
	for _, flagName := range flagNames {
		flagMap[flagName] = ""
	}
	flagsStr, _ := json.Marshal(flagMap)
	return string(flagsStr)
}

func processFlagNames(flags []string) []string {
	var resultFlags []string
	for _, flag := range flags {
		switch {
		// if flag is -abc , its equivalent to 3 short flags -a,-b, -c
		case len(flag) >= 3 && flag[0] == '-' && flag[1] != '-':
			resultFlags = append(resultFlags, strings.Split(flag[1:], "")...)
			continue
		default:
			// strip the "-" or "--" from the flag name
			resultFlags = append(resultFlags, strings.TrimLeft(flag, "-"))
		}
	}
	return resultFlags
}

func isFlagArg(arg string) bool {
	return (len(arg) >= 3 && arg[0:2] == "--") ||
		(len(arg) >= 2 && arg[0] == '-' && arg[1] != '-')
}

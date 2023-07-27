// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package common

// StringToTargetString converts string to Target type
func StringToTargetString(target string) string {
	if target == targetK8s || target == TargetK8s {
		return TargetK8s
	} else if target == targetTMC || target == TargetTMC {
		return TargetTMC
	} else if target == TargetGlobal {
		return TargetGlobal
	} else if target == TargetUnknown {
		return TargetUnknown
	}
	return TargetUnknown
}

// IsValidTarget validates the target string specified is valid or not
// TargetGlobal and TargetUnknown are special targets and hence this function
// provide flexibility additional arguments to allow them based on the requirement
func IsValidTarget(target string, allowGlobal, allowUnknown bool) bool {
	return target == targetK8s ||
		target == TargetK8s ||
		target == targetTMC ||
		target == TargetTMC ||
		(allowGlobal && target == TargetGlobal) ||
		(allowUnknown && target == TargetUnknown)
}

// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package common

// StringToTargetString converts string to Target type
func StringToTargetString(target string) string {
	if target == string(targetK8s) || target == string(TargetK8s) {
		return TargetK8s
	} else if target == string(targetTMC) || target == string(TargetTMC) {
		return TargetTMC
	} else if target == string(TargetGlobal) {
		return TargetGlobal
	} else if target == string(TargetUnknown) {
		return TargetUnknown
	}
	return TargetUnknown
}

// IsValidTarget validates the target string specified is valid or not
// TargetGlobal and TargetUnknown are special targets and hence this function
// provide flexibility additional arguments to allow them based on the requirement
func IsValidTarget(target string, allowGlobal, allowUnknown bool) bool {
	return target == string(targetK8s) ||
		target == string(TargetK8s) ||
		target == string(targetTMC) ||
		target == string(TargetTMC) ||
		(allowGlobal && target == string(TargetGlobal)) ||
		(allowUnknown && target == string(TargetUnknown))
}

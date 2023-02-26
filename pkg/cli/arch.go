// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	// MinOSArch defines minimum OS/ARCH combination for which plugin needs to be built
	MinOSArch = []Arch{LinuxAMD64, DarwinAMD64, WinAMD64}

	// AllOSArch defines all OS/ARCH combination for which plugin can be built
	AllOSArch = []Arch{LinuxAMD64, DarwinAMD64, WinAMD64, DarwinARM64, LinuxARM64}
)

// Arch represents a system architecture.
type Arch string

// BuildArch returns compile time build arch or locates it.
func BuildArch() Arch {
	return Arch(fmt.Sprintf("%s_%s", runtime.GOOS, runtime.GOARCH))
}

// IsWindows tells if an arch is windows.
func (a Arch) IsWindows() bool {
	if a == Win386 || a == WinAMD64 {
		return true
	}
	return false
}

// OS returns os-name based on the arch
func (a Arch) OS() string {
	ele := strings.Split(a.String(), "_")
	if len(ele) != 2 {
		return ""
	}
	return ele[0]
}

// Arch returns arch-name
func (a Arch) Arch() string {
	ele := strings.Split(a.String(), "_")
	if len(ele) != 2 {
		return ""
	}
	return ele[1]
}

// String converts arch to string
func (a Arch) String() string {
	return string(a)
}

const (
	// Linux386 arch.
	Linux386 Arch = "linux_386"
	// LinuxAMD64 arch.
	LinuxAMD64 Arch = "linux_amd64"
	// LinuxARM64 arch.
	LinuxARM64 Arch = "linux_arm64"
	// DarwinAMD64 arch.
	DarwinAMD64 Arch = "darwin_amd64"
	// DarwinARM64 arch.
	DarwinARM64 Arch = "darwin_arm64"
	// Win386 arch.
	Win386 Arch = "windows_386"
	// WinAMD64 arch.
	WinAMD64 Arch = "windows_amd64"
)

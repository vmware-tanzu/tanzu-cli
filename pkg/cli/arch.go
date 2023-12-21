// Copyright 2021 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package cli

import (
	"fmt"
	"runtime"
	"strings"
)

var (
	// MinOSArch defines the minimum OS/ARCH combinations for which plugins need to be built
	MinOSArch = []Arch{LinuxAMD64, DarwinAMD64, WinAMD64}

	// AllOSArch defines all OS/ARCH combinations for which plugins can be built
	AllOSArch = []Arch{LinuxAMD64, DarwinAMD64, WinAMD64, LinuxARM64, DarwinARM64, WinARM64}

	// GOOS is the current go os.  Defaults to runtime.GOOS but could be overridden.
	// The CLI code should always this variable instead of runtime.GOOS.
	GOOS = runtime.GOOS
	// GOARCH is the current go architecture.  Defaults to runtime.GOARCH but is overridden
	// for scenarios like installing AMD64 plugins on an ARM64 machine using emulation.
	// The CLI code should always this variable instead of runtime.GOARCH.
	GOARCH = runtime.GOARCH
)

// Arch represents a system architecture.
type Arch string

// BuildArch returns compile time build arch or locates it.
func BuildArch() Arch {
	return Arch(fmt.Sprintf("%s_%s", GOOS, GOARCH))
}

func SetArch(a Arch) {
	GOOS = a.OS()
	GOARCH = a.Arch()
}

// IsWindows tells if an arch is windows.
func (a Arch) IsWindows() bool {
	if a == Win386 || a == WinAMD64 || a == WinARM64 {
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
	// WinARM64 arch.
	WinARM64 Arch = "windows_arm64"
)

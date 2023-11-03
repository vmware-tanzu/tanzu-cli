// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package framework

// ContainerRuntime has operations to perform on container runtime
type ContainerRuntime interface {
	StartContainerRuntime() (stdOut, stdErr string, err error)
	ContainerRuntimeStatus() (stdOut, stdErr string, err error)
	StopContainerRuntime() (stdOut, stdErr string, err error)
}

// Docker is the container runtime of type docker
type Docker interface {
	ContainerRuntime
}

// Docker is the implementation of ContainerRuntime for docker specific
type docker struct {
	CmdOps
}

func NewDocker() Docker {
	return &docker{
		CmdOps: NewCmdOps(),
	}
}

// StartContainerRuntime starts docker daemon if not already running
func (dc *docker) StartContainerRuntime() (stdOut, stdErr string, err error) {
	if so, sb, err := dc.ContainerRuntimeStatus(); err == nil {
		return so, sb, nil
	}
	stdOutBuff, stdErrBuff, err := dc.Exec(StartDockerUbuntu)
	if err != nil {
		return stdOutBuff.String(), stdErrBuff.String(), err
	}
	return stdOutBuff.String(), stdErrBuff.String(), err
}

// ContainerRuntimeStatus returns docker daemon daemon status
func (dc *docker) ContainerRuntimeStatus() (stdOut, stdErr string, err error) {
	stdOutBuff, stdErrBuff, err := dc.Exec(DockerInfo)
	if err != nil {
		return stdOutBuff.String(), stdErrBuff.String(), err
	}
	return stdOutBuff.String(), stdErrBuff.String(), err
}

// StopContainerRuntime returns docker daemon daemon status
func (dc *docker) StopContainerRuntime() (stdOut, stdErr string, err error) {
	stdOutBuff, stdErrBuff, err := dc.Exec(StopDockerUbuntu)
	if err != nil {
		return stdOutBuff.String(), stdErrBuff.String(), err
	}
	return stdOutBuff.String(), stdErrBuff.String(), err
}

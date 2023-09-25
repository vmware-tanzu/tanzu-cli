// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

// Package wcp provides helpers to interact with a vSphere Supervisor.
package wcp

import (
	"fmt"
	"net/http"

	"github.com/vmware-tanzu/tanzu-plugin-runtime/log"
)

const (
	SupervisorVIPConfigMapName = "vip-cluster-info"
)

// IsVSphereSupervisor probes the given endpoint on a well known vSphere
// Supervisor 'login banner' endpoint.
// Returns true iff it was able to successfully determine that the endpoint was
// a vSphere Supervisor.
func IsVSphereSupervisor(endpoint string, httpClient *http.Client) (bool, error) {
	loginBannerURL := fmt.Sprintf("%s/wcp/loginbanner", endpoint)

	req, _ := http.NewRequest("GET", loginBannerURL, http.NoBody) //nolint:noctx //should rewrite http.NewRequestWithContext or add (*Request).WithContext (noctx)

	resp, err := httpClient.Do(req)
	if err != nil {
		log.Infof("Failed to test for vSphere supervisor: %+v", err)
		return false, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		log.V(4).Info("Got login banner from server")
		return true, nil
	}

	log.Infof("Could not get login banner from server, response code = %+v", resp.StatusCode)
	return false, nil
}

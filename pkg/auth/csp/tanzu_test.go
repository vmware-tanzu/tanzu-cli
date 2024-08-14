// Copyright 2023 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package csp

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

const fakeOrgName = "TestOrg"

func TestGetOrgNameFromOrgID(t *testing.T) {
	// Mock HTTP server for org name request
	server, cleanupServer := createFakeIssuerToServeOrgName()
	defer cleanupServer()

	// Mock HTTP client to use the server
	httpRestClient = &http.Client{
		Transport: http.DefaultTransport,
	}

	// Test the success path
	orgName, err := GetOrgNameFromOrgID("org123", "access123", server.URL)
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if orgName != fakeOrgName {
		t.Errorf("Expected org name '%s', got %s", fakeOrgName, orgName)
	}

	// Test the invalid org
	_, err = GetOrgNameFromOrgID("InvalidOrg", "access123", server.URL)
	assert.ErrorContains(t, err, "failed to obtain the CSP organization information")
}

// createFakeIssuerToServeOrgName creates the fake issuer to server API that return the organization information.
// It returns org name if the request is for orgID "org123" else it returns http 404 error
func createFakeIssuerToServeOrgName() (*httptest.Server, func()) {
	// Mock HTTP server for org name request
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/orgs/org123" {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(fmt.Sprintf("{\"displayName\": \"%s\"}", fakeOrgName)))
		} else {
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	return server, func() {
		server.Close()
	}
}

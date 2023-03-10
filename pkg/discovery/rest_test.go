// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package discovery

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"

	cliv1alpha1 "github.com/vmware-tanzu/tanzu-cli/apis/cli/v1alpha1"
)

const (
	basePath      = "/v1alpha1/cli/plugins"
	discoveryName = "test"
)

var (
	pluginFoo = Plugin{
		Name:               "foo",
		Description:        "A plugin for Foo",
		RecommendedVersion: "1.0.0",
		Artifacts: map[string]cliv1alpha1.ArtifactList{
			"0.0.1": {
				{
					URI:    "https://storage.googleapis.com/storage/v1/b/tanzu-plugins/o/foo-0.0.1-darwin-amd64",
					Digest: "test digest",
					OS:     "darwin",
					Arch:   "amd64",
				},
				{
					URI:    "https://storage.googleapis.com/storage/v1/b/tanzu-plugins/o/foo-0.0.1-linux-amd64",
					Digest: "test digest",
					OS:     "linux",
					Arch:   "amd64",
				},
				{
					URI:    "https://storage.googleapis.com/storage/v1/b/tanzu-plugins/o/foo-0.0.1-windows-amd64.exe",
					Digest: "test digest",
					OS:     "windows",
					Arch:   "amd64",
				},
			},
			"1.0.0": {
				{
					URI:    "https://storage.googleapis.com/storage/v1/b/tanzu-plugins/o/foo-1.0.0-darwin-amd64",
					Digest: "test digest",
					OS:     "darwin",
					Arch:   "amd64",
				},
				{
					URI:    "https://storage.googleapis.com/storage/v1/b/tanzu-plugins/o/foo-1.0.0-linux-amd64",
					Digest: "test digest",
					OS:     "linux",
					Arch:   "amd64",
				},
				{
					URI:    "https://storage.googleapis.com/storage/v1/b/tanzu-plugins/o/foo-1.0.0-windows-amd64.exe",
					Digest: "test digest",
					OS:     "windows",
					Arch:   "amd64",
				},
			},
		},
		Optional: false,
	}
	pluginBar = Plugin{
		Name:               "bar",
		Description:        "A plugin for Bar",
		RecommendedVersion: "0.0.1",
		Artifacts: map[string]cliv1alpha1.ArtifactList{
			"0.0.1": {
				{
					URI:    "https://storage.googleapis.com/storage/v1/b/tanzu-plugins/o/bar-0.0.1-darwin-amd64",
					Digest: "test digest",
					OS:     "darwin",
					Arch:   "amd64",
				},
				{
					URI:    "https://storage.googleapis.com/storage/v1/b/tanzu-plugins/o/bar-0.0.1-linux-amd64",
					Digest: "test digest",
					OS:     "linux",
					Arch:   "amd64",
				},
				{
					URI:    "https://storage.googleapis.com/storage/v1/b/tanzu-plugins/o/bar-0.0.1-windows-amd64.exe",
					Digest: "test digest",
					OS:     "windows",
					Arch:   "amd64",
				},
			},
		},
		Optional: true,
	}
	// I has has happened that some test endpoints have returned such output
	// The CLI should protect against it
	pluginEmpty = Plugin{
		Artifacts: map[string]cliv1alpha1.ArtifactList{
			"0.0.1": {
				{
					Digest: "test digest",
				},
				{
					Digest: "test digest",
				},
				{
					Digest: "test digest",
				},
			},
		},
	}
	validPlugins   = []Plugin{pluginFoo, pluginBar}
	invalidPlugins = []Plugin{pluginEmpty}
)

func createTestServer(plugins []Plugin) *httptest.Server {
	m := mux.NewRouter()
	m.HandleFunc(basePath, func(w http.ResponseWriter, _ *http.Request) {
		res := ListPluginsResponse{plugins}
		b, err := json.Marshal(res)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}

		_, err = w.Write(b)
		if err != nil {
			w.WriteHeader(http.StatusInternalServerError)
		}
	})
	return httptest.NewServer(m)
}

func TestRESTDiscovery(t *testing.T) {
	s := createTestServer(validPlugins)
	defer s.Close()

	d := NewRESTDiscovery(discoveryName, s.URL, basePath)

	expList := make([]Discovered, len(validPlugins))
	for i := range validPlugins {
		p, err := DiscoveredFromREST(&validPlugins[i])
		assert.NoError(t, err)
		p.Source = discoveryName
		expList[i] = p
	}
	actList, err := d.List()
	assert.NoError(t, err)
	assert.Equal(t, expList, actList)
}

func TestRESTDiscoveryWithInvalidPlugins(t *testing.T) {
	s := createTestServer(append(validPlugins, invalidPlugins...))
	defer s.Close()

	d := NewRESTDiscovery(discoveryName, s.URL, basePath)

	// Only the valid plugins are expected
	expList := make([]Discovered, len(validPlugins))
	for i := range validPlugins {
		p, err := DiscoveredFromREST(&validPlugins[i])
		assert.NoError(t, err)
		p.Source = discoveryName
		expList[i] = p
	}
	actList, err := d.List()
	assert.NoError(t, err)
	assert.Equal(t, expList, actList)
}

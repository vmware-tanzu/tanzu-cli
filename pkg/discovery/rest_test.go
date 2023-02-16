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
	basePath = "/v1alpha1/cli/plugins"
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
	plugins = []Plugin{pluginFoo, pluginBar}
)

func createTestServer() *httptest.Server {
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
	s := createTestServer()
	defer s.Close()

	d := NewRESTDiscovery("test", s.URL, basePath)

	expList := make([]Discovered, len(plugins))
	for i := range plugins {
		p, err := DiscoveredFromREST(&plugins[i])
		assert.NoError(t, err)
		p.Source = "test"
		expList[i] = p
	}
	actList, err := d.List()
	assert.NoError(t, err)
	assert.Equal(t, expList, actList)
}
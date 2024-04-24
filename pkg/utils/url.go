// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"errors"
	"net/url"
	"strings"
)

// JoinURL joins a base URL and a relative URL intelligently, ensuring that
// there are no unnecessary or duplicate slashes. It handles URLs where the base
// URL ends with a slash and the relative URL begins with a slash.
//
// Parameters:
// baseURL: The base URL as a string.
// relativeURL: The relative URL as a string. It could start with a slash, but it's not necessary.
//
// Returns:
// A string that is the concatenation of baseURL and relativeURL, formatted correctly.
// An error if there was a problem parsing the base URL.
func JoinURL(baseURL, relativeURL string) (string, error) {
	if baseURL == "" {
		return "", errors.New("base url is empty")
	}

	var schemaNotPresent bool

	// Check if the URL has a schema
	if !strings.HasPrefix(baseURL, "http://") && !strings.HasPrefix(baseURL, "https://") {
		schemaNotPresent = true
		// If not, prepend "https://"
		baseURL = "https://" + baseURL
	}

	// Parse the base URL into a *url.URL object
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}

	// Join the baseURL and relativeURL
	parsedBaseURL = parsedBaseURL.JoinPath(relativeURL)

	if schemaNotPresent {
		// Replace the schema prefix with the original baseUrl and return the joined URL
		return strings.TrimPrefix(parsedBaseURL.String(), "https://"), nil
	}

	// Return the joined URL as a string
	return parsedBaseURL.String(), nil
}

// ContainsRegistry returns true if the specified registryHost is part of registries
func ContainsRegistry(registries []string, registryHost string) bool {
	cleanRegistryURL := func(u string) string {
		u = strings.TrimSpace(u)
		u = strings.TrimPrefix(u, "http://")
		u = strings.TrimPrefix(u, "https://")
		return strings.Split(u, "/")[0]
	}
	registryHost = cleanRegistryURL(registryHost)

	for _, reg := range registries {
		reg = cleanRegistryURL(reg)
		if strings.EqualFold(reg, registryHost) {
			return true
		}
	}
	return false
}

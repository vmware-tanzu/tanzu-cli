// Copyright 2022 VMware, Inc. All Rights Reserved.
// SPDX-License-Identifier: Apache-2.0

package utils

import (
	"net/url"
	"path"
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
func JoinURL(baseURL, relativeURL string) string {
	// Parse the base URL into a *url.URL object
	parsedBaseURL, err := url.Parse(baseURL)
	if err != nil {
		// If the base URL is not a valid URL, return empty string
		return ""
	}

	// Remove leading slash from relativeURL if it's there
	relativeURL = strings.TrimPrefix(relativeURL, "/")

	// Join the base URL path and the relative URL.
	// path.Join() takes care of removing or adding any necessary slashes.
	parsedBaseURL.Path = path.Join(parsedBaseURL.Path, relativeURL)

	// If the Opaque field of the URL isn't empty, the URL path should be joined differently. This is because the net/url package in Go treats a specified port number as an opaque component, thus modifying the way the URL is formatted.
	// 1. scheme:opaque?query#fragment
	// 2. scheme://userinfo@host/path?query#fragment
	// If u.Opaque is non-empty, String uses the first form; otherwise it uses the second form.

	if parsedBaseURL.Opaque != "" {
		return path.Join(parsedBaseURL.String(), parsedBaseURL.Path)
	}

	// Return the joined URL as a string
	return parsedBaseURL.String()
}

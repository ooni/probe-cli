// Package urlx contains URL extensions.
package urlx

import (
	"net/url"
)

// ResolveReference constructs a new URL consisting of the given base URL with
// the path appended to the given path and the optional query.
//
// For example, given:
//
//	URL := "https://api.ooni.io/api/v1"
//	path := "/measurement_meta"
//	rawQuery := "full=true"
//
// This function will return:
//
//	result := "https://api.ooni.io/api/v1/measurement_meta?full=true"
//
// This function fails when we cannot parse URL as a [*net.URL].
func ResolveReference(baseURL, path, rawQuery string) (string, error) {
	parsedBase, err := url.Parse(baseURL)
	if err != nil {
		return "", err
	}
	ref := &url.URL{
		Path:     path,
		RawQuery: rawQuery,
	}
	return parsedBase.ResolveReference(ref).String(), nil
}

package nettests

import (
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/engine/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

// stringListToModelURLInfo is an utility function to convert
// a list of strings containing URLs into a list of model.URLInfo
// which would have been returned by an hypothetical backend
// API serving input for a test for which we don't have an API
// yet (e.g., stunreachability and dnscheck).
func stringListToModelURLInfo(input []string) (output []model.URLInfo, err error) {
	for _, URL := range input {
		if _, err = url.Parse(URL); err != nil {
			return nil, err
		}
		output = append(output, model.URLInfo{
			CategoryCode: "MISC", // hard to find a category
			CountryCode:  "XX",   // representing no country
			URL:          URL,
		})
	}
	return
}

// mustStringListToModelURLInfo is a stringListToModelURLInfo
// that calls panic in case there is an error.
func mustStringListToModelURLInfo(input []string) []model.URLInfo {
	output, err := stringListToModelURLInfo(input)
	runtimex.PanicOnError(err, "stringListToModelURLInfo failed")
	return output
}

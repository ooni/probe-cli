package httpclientx

//
// getxml.go - GET an XML response.
//

import (
	"context"
	"encoding/xml"
)

// GetXML sends a GET request and reads an XML response.
//
// Arguments:
//
// - ctx is the cancellable context;
//
// - config is the config to use;
//
// - URL is the URL to use.
//
// This function either returns an error or a valid Output.
func GetXML[Output any](ctx context.Context, config *Config, URL string) (Output, error) {
	// read the raw body
	rawrespbody, err := GetRaw(ctx, config, URL)

	// handle the case of error
	if err != nil {
		return zeroValue[Output](), err
	}

	// parse the response body as JSON
	var output Output
	if err := xml.Unmarshal(rawrespbody, &output); err != nil {
		return zeroValue[Output](), err
	}

	return output, nil
}

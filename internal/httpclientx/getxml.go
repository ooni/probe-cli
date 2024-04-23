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
// - URL is the URL to use;
//
// - config is the config to use.
//
// This function either returns an error or a valid Output.
func GetXML[Output any](ctx context.Context, URL string, config *Config) (Output, error) {
	return NewOverlappedGetXML[Output](config).Run(ctx, URL)
}

func getXML[Output any](ctx context.Context, URL string, config *Config) (Output, error) {
	// read the raw body
	rawrespbody, err := GetRaw(ctx, URL, config)

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

package httpclientx

//
// getjson.go - GET a JSON response.
//

import (
	"context"
	"encoding/json"
)

// GetJSON sends a GET request and reads a JSON response.
//
// Arguments:
//
// - ctx is the cancellable context;
//
// - config contains the config;
//
// - URL is the URL to use.
//
// This function either returns an error or a valid Output.
func GetJSON[Output any](ctx context.Context, config *Config, URL string) (Output, error) {
	return NewOverlappedGetJSON[Output](config).Run(ctx, URL)
}

func getJSON[Output any](ctx context.Context, config *Config, URL string) (Output, error) {
	// read the raw body
	rawrespbody, err := GetRaw(ctx, config, URL)

	// handle the case of error
	if err != nil {
		return zeroValue[Output](), err
	}

	// parse the response body as JSON
	var output Output
	if err := json.Unmarshal(rawrespbody, &output); err != nil {
		return zeroValue[Output](), err
	}

	return output, nil
}

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
// - URL is the URL to use;
//
// - config contains the config.
//
// This function either returns an error or a valid Output.
func GetJSON[Output any](ctx context.Context, URL string, config *Config) (Output, error) {
	return NewOverlappedGetJSON[Output](config).Run(ctx, URL)
}

func getJSON[Output any](ctx context.Context, URL string, config *Config) (Output, error) {
	// read the raw body
	rawrespbody, err := GetRaw(ctx, URL, config)

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

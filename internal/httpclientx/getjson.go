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
// - base is the HTTP [*BaseURL] to use;
//
// - config contains the config.
//
// This function either returns an error or a valid Output.
func GetJSON[Output any](ctx context.Context, base *BaseURL, config *Config) (Output, error) {
	return OverlappedIgnoreIndex(NewOverlappedGetJSON[Output](config).Run(ctx, base))
}

func getJSON[Output any](ctx context.Context, base *BaseURL, config *Config) (Output, error) {
	// read the raw body
	rawrespbody, err := GetRaw(ctx, base, config)

	// handle the case of error
	if err != nil {
		return zeroValue[Output](), err
	}

	// parse the response body as JSON
	var output Output
	if err := json.Unmarshal(rawrespbody, &output); err != nil {
		return zeroValue[Output](), err
	}

	// avoid returning nil pointers, maps, slices
	return NilSafetyErrorIfNil(output)
}

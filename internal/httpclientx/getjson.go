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
// - epnt is the HTTP [*Endpoint] to use;
//
// - config contains the config.
//
// This function either returns an error or a valid Output.
func GetJSON[Output any](ctx context.Context, epnt *Endpoint, config *Config) (Output, error) {
	return OverlappedIgnoreIndex(NewOverlappedGetJSON[Output](config).Run(ctx, epnt))
}

func getJSON[Output any](ctx context.Context, epnt *Endpoint, config *Config) (Output, error) {
	// read the raw body
	rawrespbody, err := GetRaw(ctx, epnt, config)

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

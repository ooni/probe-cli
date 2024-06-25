package httpclientx

//
// postjson.go - POST a JSON request and read a JSON response.
//

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
)

// PostJSON sends a POST request with a JSON body and reads a JSON response.
//
// Arguments:
//
// - ctx is the cancellable context;
//
// - epnt is the HTTP [*Endpoint] to use;
//
// - input is the input structure to JSON serialize as the request body;
//
// - config is the config to use.
//
// This function either returns an error or a valid Output.
func PostJSON[Input, Output any](ctx context.Context, epnt *Endpoint, input Input, config *Config) (Output, error) {
	return OverlappedIgnoreIndex(NewOverlappedPostJSON[Input, Output](input, config).Run(ctx, epnt))
}

func postJSON[Input, Output any](ctx context.Context, epnt *Endpoint, input Input, config *Config) (Output, error) {
	// ensure we're not sending a nil map, pointer, or slice
	if _, err := NilSafetyErrorIfNil(input); err != nil {
		return zeroValue[Output](), err
	}

	// serialize the request body
	rawreqbody, err := json.Marshal(input)
	if err != nil {
		return zeroValue[Output](), err
	}

	// log the raw request body
	config.Logger.Debugf("POST %s: raw request body: %s", epnt.URL, string(rawreqbody))

	// construct the request to use
	req, err := http.NewRequestWithContext(ctx, "POST", epnt.URL, bytes.NewReader(rawreqbody))
	if err != nil {
		return zeroValue[Output](), err
	}

	// assign the content type
	req.Header.Set("Content-Type", "application/json")

	// get the raw response body
	rawrespbody, err := do(ctx, req, epnt, config)

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

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
// - config is the config to use;
//
// - URL is the URL to use;
//
// - input is the input structure to JSON serialize as the request body.
//
// This function either returns an error or a valid Output.
func PostJSON[Input, Output any](ctx context.Context, config *Config, URL string, input Input) (Output, error) {
	return NewOverlappedPostJSON[Input, Output](config, input).Run(ctx, URL)
}

func postJSON[Input, Output any](ctx context.Context, config *Config, URL string, input Input) (Output, error) {
	// serialize the request body
	rawreqbody, err := json.Marshal(input)
	if err != nil {
		return zeroValue[Output](), err
	}

	// log the raw request body
	config.Logger.Debugf("POST %s: raw request body: %s", URL, string(rawreqbody))

	// construct the request to use
	req, err := http.NewRequestWithContext(ctx, "POST", URL, bytes.NewReader(rawreqbody))
	if err != nil {
		return zeroValue[Output](), err
	}

	// assign the content type
	req.Header.Set("Content-Type", "application/json")

	// get the raw response body
	rawrespbody, err := do(ctx, req, config)

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

package httpclientx

//
// getraw.go - GET a raw response.
//

import (
	"context"
	"net/http"
)

// GetRaw sends a GET request and reads a raw response.
//
// Arguments:
//
// - ctx is the cancellable context;
//
// - base is the HTTP [*BaseURL] to use;
//
// - config is the config to use.
//
// This function either returns an error or a valid Output.
func GetRaw(ctx context.Context, base *BaseURL, config *Config) ([]byte, error) {
	return OverlappedIgnoreIndex(NewOverlappedGetRaw(config).Run(ctx, base))
}

func getRaw(ctx context.Context, base *BaseURL, config *Config) ([]byte, error) {
	// construct the request to use
	req, err := http.NewRequestWithContext(ctx, "GET", base.Value, nil)
	if err != nil {
		return nil, err
	}

	// get raw response body
	return do(ctx, req, base, config)
}

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
// - config is the config to use;
//
// - URL is the URL to use.
//
// This function either returns an error or a valid Output.
func GetRaw(ctx context.Context, URL string, config *Config) ([]byte, error) {
	return NewOverlappedGetRaw(config).Run(ctx, URL)
}

func getRaw(ctx context.Context, URL string, config *Config) ([]byte, error) {
	// construct the request to use
	req, err := http.NewRequestWithContext(ctx, "GET", URL, nil)
	if err != nil {
		return nil, err
	}

	// get raw response body
	return do(ctx, req, config)
}

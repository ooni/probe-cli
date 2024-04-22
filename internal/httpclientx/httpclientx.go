// Package httpclientx contains extensions to more easily invoke HTTP APIs.
package httpclientx

//
// httpclientx.go - common code
//

import (
	"compress/gzip"
	"context"
	"io"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// ErrRequestFailed indicates that an HTTP request status indicates failure.
type ErrRequestFailed struct {
	StatusCode int
}

var _ error = &ErrRequestFailed{}

// Error returns the error as a string.
//
// The string returned by this error starts with the httpx prefix for backwards
// compatibility with the legacy httpx package.
func (err *ErrRequestFailed) Error() string {
	return "httpx: request failed"
}

// zeroValue is a convenience function to return the zero value.
func zeroValue[T any]() T {
	return *new(T)
}

// newLimitReader is a wrapper for [io.LimitReader] that automatically
// sets the maximum readable amount of bytes.
func newLimitReader(r io.Reader) io.Reader {
	return io.LimitReader(r, 1<<24)
}

// do is the internal function to finish preparing the request and getting a raw response.
func do(ctx context.Context, req *http.Request, config *Config) ([]byte, error) {
	// optionally assign authorization
	if value := config.Authorization; value != "" {
		req.Header.Set("Authorization", value)
	}

	// assign the user agent
	req.Header.Set("User-Agent", config.UserAgent)

	// say that we're accepting gzip encoded bodies
	req.Header.Set("Accept-Encoding", "gzip")

	// get the response
	resp, err := config.Client.Do(req)

	// handle the case of error
	if err != nil {
		return nil, err
	}

	// eventually close the response body
	defer resp.Body.Close()

	// Implementation note: here we choose to always read the response
	// body before checking the status code because it helps a lot to log
	// the response body received on failure when testing a backend

	var baseReader io.Reader = resp.Body

	// handle the case of gzip encoded body
	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzreader, err := gzip.NewReader(baseReader)
		if err != nil {
			return nil, err
		}
		baseReader = gzreader
	}

	// protect against unreasonably large response bodies
	limitReader := newLimitReader(baseReader)

	// read the raw body
	rawrespbody, err := netxlite.ReadAllContext(ctx, limitReader)

	// handle the case of failure
	if err != nil {
		return nil, err
	}

	// log the response body for debugging purposes
	config.Logger.Debugf("%s %s: raw response body: %s", req.Method, req.URL.String(), string(rawrespbody))

	// handle the case of HTTP error
	if resp.StatusCode != 200 {
		return nil, &ErrRequestFailed{resp.StatusCode}
	}

	return rawrespbody, nil
}

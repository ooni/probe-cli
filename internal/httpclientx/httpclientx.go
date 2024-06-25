// Package httpclientx contains extensions to more easily invoke HTTP APIs.
package httpclientx

//
// httpclientx.go - common code
//

import (
	"compress/gzip"
	"context"
	"errors"
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

// ErrTruncated indicates we truncated the response body.
//
// Note: we SHOULD NOT change the error string because this error string was previously
// used by the httpapi package and it's better to keep the same strings.
var ErrTruncated = errors.New("httpapi: truncated response body")

// do is the internal function to finish preparing the request and getting a raw response.
func do(ctx context.Context, req *http.Request, epnt *Endpoint, config *Config) ([]byte, error) {
	// optionally assign authorization
	if value := config.Authorization; value != "" {
		req.Header.Set("Authorization", value)
	}

	// assign the user agent
	req.Header.Set("User-Agent", config.UserAgent)

	// say that we're accepting gzip encoded bodies
	req.Header.Set("Accept-Encoding", "gzip")

	// OPTIONALLY allow for cloudfronting (the default in net/http is for
	// the req.Host to be empty and to use req.URL.Host)
	req.Host = epnt.Host

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
	//
	// read one more byte than the maximum allowed size so we can
	// always tell whether it was truncated here
	limitReader := io.LimitReader(baseReader, config.maxResponseBodySize()+1)

	// read the raw body
	rawrespbody, err := netxlite.ReadAllContext(ctx, limitReader)

	// handle the case of failure
	if err != nil {
		return nil, err
	}

	// handle the case of truncated body
	if int64(len(rawrespbody)) > config.maxResponseBodySize() {
		return nil, ErrTruncated
	}

	// log the response body for debugging purposes
	config.Logger.Debugf("%s %s: raw response body: %s", req.Method, req.URL.String(), string(rawrespbody))

	// handle the case of HTTP error
	if resp.StatusCode != 200 {
		return nil, &ErrRequestFailed{resp.StatusCode}
	}

	// make sure we replace a nil slice with an empty slice
	return NilSafetyAvoidNilBytesSlice(rawrespbody), nil
}

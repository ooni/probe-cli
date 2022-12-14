package httpapi

//
// Calling HTTP APIs.
//

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// joinURLPath appends |resourcePath| to |urlPath|.
func joinURLPath(urlPath, resourcePath string) string {
	if resourcePath == "" {
		if urlPath == "" {
			return "/"
		}
		return urlPath
	}
	if !strings.HasSuffix(urlPath, "/") {
		urlPath += "/"
	}
	resourcePath = strings.TrimPrefix(resourcePath, "/")
	return urlPath + resourcePath
}

// newRequest creates a new http.Request from the given |ctx|, |endpoint|, and |desc|.
func newRequest(ctx context.Context, endpoint *Endpoint, desc *Descriptor) (*http.Request, error) {
	URL, err := url.Parse(endpoint.BaseURL)
	if err != nil {
		return nil, err
	}
	// BaseURL and resource URL are joined if they have a path
	URL.Path = joinURLPath(URL.Path, desc.URLPath)
	if len(desc.URLQuery) > 0 {
		URL.RawQuery = desc.URLQuery.Encode()
	} else {
		URL.RawQuery = "" // as documented we only honour desc.URLQuery
	}
	var reqBody io.Reader
	if len(desc.RequestBody) > 0 {
		reqBody = bytes.NewReader(desc.RequestBody)
		endpoint.Logger.Debugf("httpapi: request body length: %d", len(desc.RequestBody))
		if desc.LogBody {
			endpoint.Logger.Debugf("httpapi: request body: %s", string(desc.RequestBody))
		}
	}
	request, err := http.NewRequestWithContext(ctx, desc.Method, URL.String(), reqBody)
	if err != nil {
		return nil, err
	}
	request.Host = endpoint.Host // allow cloudfronting
	if desc.Authorization != "" {
		request.Header.Set("Authorization", desc.Authorization)
	}
	if desc.ContentType != "" {
		request.Header.Set("Content-Type", desc.ContentType)
	}
	if desc.Accept != "" {
		request.Header.Set("Accept", desc.Accept)
	}
	if endpoint.UserAgent != "" {
		request.Header.Set("User-Agent", endpoint.UserAgent)
	}
	if desc.AcceptEncodingGzip {
		request.Header.Set("Accept-Encoding", "gzip")
	}
	return request, nil
}

// ErrHTTPRequestFailed indicates that the server returned >= 400.
type ErrHTTPRequestFailed struct {
	// StatusCode is the status code that failed.
	StatusCode int
}

// Error implements error.
func (err *ErrHTTPRequestFailed) Error() string {
	return fmt.Sprintf("httpapi: http request failed: %d", err.StatusCode)
}

// errMaybeCensorship indicates that there was an error at the networking layer
// including, e.g., DNS, TCP connect, TLS. When we see this kind of error, we
// will consider retrying with another endpoint under the assumption that it
// may be that the current endpoint is censored.
type errMaybeCensorship struct {
	// Err is the underlying error
	Err error
}

// Error implements error
func (err *errMaybeCensorship) Error() string {
	return err.Err.Error()
}

// Unwrap allows to get the underlying error
func (err *errMaybeCensorship) Unwrap() error {
	return err.Err
}

// ErrTruncated indicates we truncated the response body.
var ErrTruncated = errors.New("httpapi: truncated response body")

// docall calls the API represented by the given request |req| on the given |endpoint|
// and returns the response and its body or an error.
func docall(endpoint *Endpoint, desc *Descriptor, request *http.Request) (*http.Response, []byte, error) {
	// Implementation note: remember to mark errors for which you want
	// to retry with another endpoint using errMaybeCensorship.

	response, err := endpoint.HTTPClient.Do(request)
	if err != nil {
		return nil, nil, &errMaybeCensorship{err}
	}
	defer response.Body.Close()

	var reader io.Reader = response.Body
	if response.Header.Get("Content-Encoding") == "gzip" {
		reader, err = gzip.NewReader(reader)
		if err != nil {
			// This case happens when we cannot read the gzip header
			// hence it can be "triggered" remotely and we cannot just
			// panic on error to handle this error condition.
			return response, nil, err
		}
	}
	maxBodySize := desc.MaxBodySize
	if maxBodySize <= 0 {
		maxBodySize = DefaultMaxBodySize
	}
	// Implementation note: when there's decompression we must (obviously?)
	// enforce the maximum body size on the _decompressed_ body.
	reader = io.LimitReader(reader, maxBodySize)

	// Implementation note: always read and log the response body _before_
	// checking the status code, since it's quite useful to log the JSON
	// returned by the OONI API in case of errors. Obviously, the flip side
	// of this choice is that we read potentially very large error pages.
	data, err := netxlite.ReadAllContext(request.Context(), reader)
	if err != nil {
		return response, nil, &errMaybeCensorship{err}
	}
	if int64(len(data)) >= maxBodySize {
		return response, nil, ErrTruncated
	}
	endpoint.Logger.Debugf("httpapi: response body length: %d bytes", len(data))
	if desc.LogBody {
		endpoint.Logger.Debugf("httpapi: response body: %s", string(data))
	}

	if response.StatusCode >= 400 {
		return response, nil, &ErrHTTPRequestFailed{response.StatusCode}
	}
	return response, data, nil
}

// call calls the given API and returns the response and the raw response body.
func call(ctx context.Context, desc *Descriptor, endpoint *Endpoint) (*http.Response, []byte, error) {
	timeout := desc.Timeout
	if timeout <= 0 {
		timeout = DefaultCallTimeout // as documented
	}
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	request, err := newRequest(ctx, endpoint, desc)
	if err != nil {
		return nil, nil, err
	}
	return docall(endpoint, desc, request)
}

// RawCall calls the API described by spec using endpoint.
//
// Note: this function returns ErrHTTPRequestFailed if the HTTP status code is
// greater or equal than 400. You could use errors.As to obtain a copy of the
// error that was returned and see for yourself the actual status code.
func RawCall(ctx context.Context, spec SimpleSpec, endpoint *Endpoint) ([]byte, error) {
	desc, err := spec.Descriptor()
	if err != nil {
		return nil, err
	}
	_, data, err := call(ctx, desc, endpoint)
	return data, err
}

// goodContentTypeForJSON tracks known-good content-types for JSON. If the content-type
// is not in this map, |CallWithJSONResponse| emits a warning message.
var goodContentTypeForJSON = map[string]bool{
	ApplicationJSON: true,
}

// TypedCall calls the API described by spec using endpoint.
//
// Note: this function returns ErrHTTPRequestFailed if the HTTP status code is
// greater or equal than 400. You could use errors.As to obtain a copy of the
// error that was returned and see for yourself the actual status code.
func TypedCall[T any](ctx context.Context, spec TypedSpec[T], endpoint *Endpoint) (*T, error) {
	desc, err := spec.Descriptor()
	if err != nil {
		return nil, err
	}
	httpResp, rawRespBody, err := call(ctx, desc, endpoint)
	if err != nil {
		return nil, err
	}
	if ctype := httpResp.Header.Get("Content-Type"); !goodContentTypeForJSON[ctype] {
		endpoint.Logger.Warnf("httpapi: unexpected content-type: %s", ctype)
		// fallthrough
	}
	value := spec.ZeroResponse()
	if err := json.Unmarshal(rawRespBody, &value); err != nil {
		return nil, err
	}
	return &value, nil
}

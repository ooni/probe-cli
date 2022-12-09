package httpapi

//
// Calling HTTP APIs.
//

import (
	"bytes"
	"context"
	"encoding/json"
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
	// Implementation note: always read and log the response body since
	// it's quite useful to see the response JSON on API error.
	r := io.LimitReader(response.Body, DefaultMaxBodySize)
	data, err := netxlite.ReadAllContext(request.Context(), r)
	if err != nil {
		return response, nil, &errMaybeCensorship{err}
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

// call is like Call but also returns the response.
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

// Call invokes the API described by |desc| on the given HTTP |endpoint| and
// returns the response body (as a slice of bytes) or an error.
//
// Note: this function returns ErrHTTPRequestFailed if the HTTP status code is
// greater or equal than 400. You could use errors.As to obtain a copy of the
// error that was returned and see for yourself the actual status code.
func Call(ctx context.Context, desc *Descriptor, endpoint *Endpoint) ([]byte, error) {
	_, rawResponseBody, err := call(ctx, desc, endpoint)
	return rawResponseBody, err
}

// goodContentTypeForJSON tracks known-good content-types for JSON. If the content-type
// is not in this map, |CallWithJSONResponse| emits a warning message.
var goodContentTypeForJSON = map[string]bool{
	applicationJSON: true,
}

// CallWithJSONResponse is like Call but also assumes that the response is a
// JSON body and attempts to parse it into the |response| field.
//
// Note: this function returns ErrHTTPRequestFailed if the HTTP status code is
// greater or equal than 400. You could use errors.As to obtain a copy of the
// error that was returned and see for yourself the actual status code.
func CallWithJSONResponse(ctx context.Context, desc *Descriptor, endpoint *Endpoint, response any) error {
	httpResp, rawRespBody, err := call(ctx, desc, endpoint)
	if err != nil {
		return err
	}
	if ctype := httpResp.Header.Get("Content-Type"); !goodContentTypeForJSON[ctype] {
		endpoint.Logger.Warnf("httpapi: unexpected content-type: %s", ctype)
		// fallthrough
	}
	return json.Unmarshal(rawRespBody, response)
}

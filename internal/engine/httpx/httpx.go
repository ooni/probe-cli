// Package httpx contains http extensions.
package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// DefaultMaxBodySize is the default value for the maximum
// body size you can fetch using an APIClient.
const DefaultMaxBodySize = 1 << 22

// APIClient is an extended HTTP client. To construct this APIClient, make
// sure you initialize all fields marked as MANDATORY.
type APIClient struct {
	// Accept contains the OPTIONAL accept header.
	Accept string

	// Authorization contains the OPTIONAL authorization header.
	Authorization string

	// BaseURL is the MANDATORY base URL of the API.
	BaseURL string

	// HTTPClient is the MANDATORY underlying http client to use.
	HTTPClient model.HTTPClient

	// Host allows to OPTIONALLY set a specific host header. This is useful
	// to implement, e.g., cloudfronting.
	Host string

	// Logger is MANDATORY the logger to use.
	Logger model.DebugLogger

	// UserAgent is the OPTIONAL user agent to use.
	UserAgent string
}

// newRequestWithJSONBody creates a new request with a JSON body
func (c *APIClient) newRequestWithJSONBody(
	ctx context.Context, method, resourcePath string,
	query url.Values, body interface{}) (*http.Request, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	c.Logger.Debugf("httpx: request body: %d bytes", len(data))
	request, err := c.newRequest(
		ctx, method, resourcePath, query, bytes.NewReader(data))
	if err != nil {
		return nil, err
	}
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	return request, nil
}

// newRequest creates a new request.
func (c *APIClient) newRequest(ctx context.Context, method, resourcePath string,
	query url.Values, body io.Reader) (*http.Request, error) {
	URL, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}
	URL.Path = resourcePath
	if query != nil {
		URL.RawQuery = query.Encode()
	}
	c.Logger.Debugf("httpx: method: %s", method)
	c.Logger.Debugf("httpx: URL: %s", URL.String())
	request, err := http.NewRequest(method, URL.String(), body)
	if err != nil {
		return nil, err
	}
	request.Host = c.Host // allow cloudfronting
	if c.Authorization != "" {
		request.Header.Set("Authorization", c.Authorization)
	}
	if c.Accept != "" {
		request.Header.Set("Accept", c.Accept)
	}
	request.Header.Set("User-Agent", c.UserAgent)
	return request.WithContext(ctx), nil
}

// ErrRequestFailed indicates that the server returned >= 400.
var ErrRequestFailed = errors.New("httpx: request failed")

// do performs the provided request and returns the response body or an error.
func (c *APIClient) do(request *http.Request) ([]byte, error) {
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: %s", ErrRequestFailed, response.Status)
	}
	r := io.LimitReader(response.Body, DefaultMaxBodySize)
	data, err := netxlite.ReadAllContext(request.Context(), r)
	if err != nil {
		return nil, err
	}
	return data, nil
}

// doJSON performs the provided request and unmarshals the JSON response body
// into the provided output variable.
func (c *APIClient) doJSON(request *http.Request, output interface{}) error {
	data, err := c.do(request)
	if err != nil {
		return err
	}
	c.Logger.Debugf("httpx: response body: %d bytes", len(data))
	return json.Unmarshal(data, output)
}

// GetJSON reads the JSON resource at resourcePath and unmarshals the
// results into output. The request is bounded by the lifetime of the
// context passed as argument. Returns the error that occurred.
func (c *APIClient) GetJSON(ctx context.Context, resourcePath string, output interface{}) error {
	return c.GetJSONWithQuery(ctx, resourcePath, nil, output)
}

// GetJSONWithQuery is like GetJSON but also has a query.
func (c *APIClient) GetJSONWithQuery(
	ctx context.Context, resourcePath string,
	query url.Values, output interface{}) error {
	request, err := c.newRequest(ctx, "GET", resourcePath, query, nil)
	if err != nil {
		return err
	}
	return c.doJSON(request, output)
}

// PostJSON creates a JSON subresource of the resource at resourcePath
// using the JSON document at input and returning the result into the
// JSON document at output. The request is bounded by the context's
// lifetime. Returns the error that occurred.
func (c *APIClient) PostJSON(
	ctx context.Context, resourcePath string, input, output interface{}) error {
	request, err := c.newRequestWithJSONBody(ctx, "POST", resourcePath, nil, input)
	if err != nil {
		return err
	}
	return c.doJSON(request, output)
}

// FetchResource fetches the specified resource and returns it.
func (c *APIClient) FetchResource(ctx context.Context, URLPath string) ([]byte, error) {
	request, err := c.newRequest(ctx, "GET", URLPath, nil, nil)
	if err != nil {
		return nil, err
	}
	return c.do(request)
}

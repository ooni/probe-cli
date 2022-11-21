// Package httpx contains http extensions.
//
// Deprecated: new code should use httpapi instead. While this package and httpapi
// are basically using the same implementation, the API exposed by httpapi allows
// us to try the same request with multiple HTTP endpoints.
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
	"strings"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// APIClientTemplate is a template for constructing an APIClient.
type APIClientTemplate struct {
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

	// LogBody is the OPTIONAL flag to force logging the bodies.
	LogBody bool

	// Logger is MANDATORY the logger to use.
	Logger model.DebugLogger

	// UserAgent is the OPTIONAL user agent to use.
	UserAgent string
}

// WithBodyLogging enables logging of request and response bodies.
func (tmpl *APIClientTemplate) WithBodyLogging() *APIClientTemplate {
	out := APIClientTemplate(*tmpl)
	out.LogBody = true
	return &out
}

// Build creates an APIClient from the APIClientTemplate.
func (tmpl *APIClientTemplate) Build() APIClient {
	return tmpl.BuildWithAuthorization(tmpl.Authorization)
}

// BuildWithAuthorization creates an APIClient from the
// APIClientTemplate and ensures it uses the given authorization
// value for APIClient.Authorization in subsequent API calls.
func (tmpl *APIClientTemplate) BuildWithAuthorization(authorization string) APIClient {
	ac := apiClient(*tmpl)
	ac.Authorization = authorization
	return &ac
}

// DefaultMaxBodySize is the default value for the maximum
// body size you can fetch using an APIClient.
const DefaultMaxBodySize = 1 << 22

// APIClient is a client configured to call a given API identified
// by a given baseURL and using a given model.HTTPClient.
//
// The resource path argument passed to APIClient methods is appended
// to the base URL's path for determining the full URL's path.
type APIClient interface {
	// GetJSON reads the JSON resource whose path is obtained concatenating
	// the baseURL's path with `resourcePath` and unmarshals the results
	// into `output`. The request is bounded by the lifetime of the
	// context passed as argument. Returns the error that occurred.
	GetJSON(ctx context.Context, resourcePath string, output interface{}) error

	// GetJSONWithQuery is like GetJSON but also has a query.
	GetJSONWithQuery(ctx context.Context, resourcePath string,
		query url.Values, output interface{}) error

	// PostJSON creates a JSON subresource of the resource whose
	// path is obtained concatenating the baseURL'spath with `resourcePath` using
	// the JSON document at `input` as value and returning the result into the
	// JSON document at output. The request is bounded by the context's
	// lifetime. Returns the error that occurred.
	PostJSON(ctx context.Context, resourcePath string, input, output interface{}) error

	// FetchResource fetches the specified resource and returns it.
	FetchResource(ctx context.Context, URLPath string) ([]byte, error)
}

// apiClient is an extended HTTP client. To construct this struct, make
// sure you initialize all fields marked as MANDATORY.
type apiClient struct {
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

	// LogBody is the OPTIONAL flag to force logging the bodies.
	LogBody bool

	// Logger is MANDATORY the logger to use.
	Logger model.DebugLogger

	// UserAgent is the OPTIONAL user agent to use.
	UserAgent string
}

// newRequestWithJSONBody creates a new request with a JSON body
func (c *apiClient) newRequestWithJSONBody(
	ctx context.Context, method, resourcePath string,
	query url.Values, body interface{}) (*http.Request, error) {
	data, err := json.Marshal(body)
	if err != nil {
		return nil, err
	}
	c.Logger.Debugf("httpx: request body length: %d bytes", len(data))
	if c.LogBody {
		c.Logger.Debugf("httpx: request body: %s", string(data))
	}
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

// joinURLPath appends resourcePath to the urlPath.
func (c *apiClient) joinURLPath(urlPath, resourcePath string) string {
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

// newRequest creates a new request.
func (c *apiClient) newRequest(ctx context.Context, method, resourcePath string,
	query url.Values, body io.Reader) (*http.Request, error) {
	URL, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}
	// BaseURL and resource URL is joined if they have a path
	URL.Path = c.joinURLPath(URL.Path, resourcePath)
	if query != nil {
		URL.RawQuery = query.Encode()
	}
	request, err := http.NewRequestWithContext(ctx, method, URL.String(), body)
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
	return request, nil
}

// ErrRequestFailed indicates that the server returned >= 400.
var ErrRequestFailed = errors.New("httpx: request failed")

// do performs the provided request and returns the response body or an error.
func (c *apiClient) do(request *http.Request) ([]byte, error) {
	response, err := c.HTTPClient.Do(request)
	if err != nil {
		return nil, err
	}
	defer response.Body.Close()
	// Implementation note: always read and log the response body since
	// it's quite useful to see the response JSON on API error.
	r := io.LimitReader(response.Body, DefaultMaxBodySize)
	data, err := netxlite.ReadAllContext(request.Context(), r)
	if err != nil {
		return nil, err
	}
	c.Logger.Debugf("httpx: response body length: %d bytes", len(data))
	if c.LogBody {
		c.Logger.Debugf("httpx: response body: %s", string(data))
	}
	if response.StatusCode >= 400 {
		return nil, fmt.Errorf("%w: %s", ErrRequestFailed, response.Status)
	}
	return data, nil
}

// doJSON performs the provided request and unmarshals the JSON response body
// into the provided output variable.
func (c *apiClient) doJSON(request *http.Request, output interface{}) error {
	data, err := c.do(request)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, output)
}

// GetJSON implements APIClient.GetJSON.
func (c *apiClient) GetJSON(ctx context.Context, resourcePath string, output interface{}) error {
	return c.GetJSONWithQuery(ctx, resourcePath, nil, output)
}

// GetJSONWithQuery implements APIClient.GetJSONWithQuery.
func (c *apiClient) GetJSONWithQuery(
	ctx context.Context, resourcePath string,
	query url.Values, output interface{}) error {
	request, err := c.newRequest(ctx, "GET", resourcePath, query, nil)
	if err != nil {
		return err
	}
	return c.doJSON(request, output)
}

// PostJSON implements APIClient.PostJSON.
func (c *apiClient) PostJSON(
	ctx context.Context, resourcePath string, input, output interface{}) error {
	request, err := c.newRequestWithJSONBody(ctx, "POST", resourcePath, nil, input)
	if err != nil {
		return err
	}
	return c.doJSON(request, output)
}

// FetchResource implements APIClient.FetchResource.
func (c *apiClient) FetchResource(ctx context.Context, URLPath string) ([]byte, error) {
	request, err := c.newRequest(ctx, "GET", URLPath, nil, nil)
	if err != nil {
		return nil, err
	}
	return c.do(request)
}

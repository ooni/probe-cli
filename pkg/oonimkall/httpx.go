package oonimkall

//
// HTTP eXtensions
//

import (
	"errors"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Implementation note: I am keeping this API as simple as possible. Obviously, there
// is room for improvements and possible caveats. For example:
//
// 1. we may want to send a POST request with a body (not yet implemented);
//
// 2. we may want to disable failing if status code is not 200 (not yet implemented);
//
// 3. we may want to see the response status code (not yet implemented);
//
// 4. we may want to efficiently support binary bodies (not yet implemented).
//
// If needed, we will adapt the API and implement new features.

// HTTPRequest is an HTTP request to send.
type HTTPRequest struct {
	// Method is the MANDATORY request method.
	Method string

	// Url is the MANDATORY request URL.
	//
	// This variable MUST be named "Url" not "URL"; see https://github.com/ooni/probe/issues/2701.
	Url string
}

// HTTPResponse is an HTTP response.
type HTTPResponse struct {
	// Body is the response body.
	Body string
}

// HTTPDo performs an HTTP request and returns the response.
//
// This method uses the default HTTP client of the session, which is the same
// client that the OONI engine uses to communicate with the OONI backend.
//
// This method throws an exception if the HTTP request status code is not 200.
func (sess *Session) HTTPDo(ctx *Context, jreq *HTTPRequest) (*HTTPResponse, error) {
	sess.mtx.Lock()
	defer sess.mtx.Unlock()
	return sess.httpDoLocked(ctx, jreq)
}

func (sess *Session) httpDoLocked(ctx *Context, jreq *HTTPRequest) (*HTTPResponse, error) {
	clnt := sess.sessp.DefaultHTTPClient()

	req, err := http.NewRequestWithContext(ctx.ctx, jreq.Method, jreq.Url, nil)
	if err != nil {
		return nil, err
	}

	resp, err := clnt.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, errors.New("httpx: HTTP request failed")
	}

	rawResp, err := netxlite.ReadAllContext(ctx.ctx, resp.Body)
	if err != nil {
		return nil, err
	}

	jResp := &HTTPResponse{
		Body: string(rawResp),
	}

	return jResp, nil
}

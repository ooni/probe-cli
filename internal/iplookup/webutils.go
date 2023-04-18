package iplookup

//
// Common HTTP code
//

import (
	"context"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// httpDo is the common function to issue an HTTP request and get the response body.
func (c *Client) httpDo(req *http.Request, family model.AddressFamily) ([]byte, error) {
	// honour the TestingHTTPDo hook, if needed.
	if c.testingHTTPDo != nil {
		return c.testingHTTPDo(req)
	}

	// make sure we eventually time out
	ctx := req.Context()
	ctx, cancel := context.WithTimeout(ctx, defaultTimeout)
	defer cancel()
	req = req.WithContext(ctx)

	// create HTTP client
	//
	// Note: create an address-family aware resolver to make sure we're not
	// going to use an IP addresses belongong to the wrong family.
	httpClient := netxlite.NewHTTPClientWithResolver(c.logger, c.newAddressFamilyResolver(family))
	defer httpClient.CloseIdleConnections()

	// issue HTTP request and get response
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	// make sure the request succeded
	if resp.StatusCode != 200 {
		return nil, ErrHTTPRequestFailed
	}

	// read response body
	return netxlite.ReadAllContext(req.Context(), resp.Body)
}

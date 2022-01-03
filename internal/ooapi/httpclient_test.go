package ooapi

import (
	"net/http"
	"testing"
)

type VerboseHTTPClient struct {
	T *testing.T
}

func (c *VerboseHTTPClient) Do(req *http.Request) (*http.Response, error) {
	c.T.Logf("> %s %s", req.Method, req.URL.String())
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		c.T.Logf("< %s", err.Error())
		return nil, err
	}
	c.T.Logf("< %d", resp.StatusCode)
	return resp, nil
}

func (c *VerboseHTTPClient) CloseIdleConnections() {}

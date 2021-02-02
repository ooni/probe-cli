package httpx

import (
	"context"
	"crypto/sha256"
	"fmt"
)

// FetchResource fetches the specified resource and returns it.
func (c Client) FetchResource(ctx context.Context, URLPath string) ([]byte, error) {
	request, err := c.NewRequest(ctx, "GET", URLPath, nil, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(request)
}

// FetchResourceAndVerify fetches and verifies a specific resource.
func (c Client) FetchResourceAndVerify(ctx context.Context, URL, SHA256Sum string) ([]byte, error) {
	c.Logger.Debugf("httpx: expected SHA256: %s", SHA256Sum)
	data, err := c.FetchResource(ctx, URL)
	if err != nil {
		return nil, err
	}
	s := fmt.Sprintf("%x", sha256.Sum256(data))
	c.Logger.Debugf("httpx: real SHA256: %s", s)
	if SHA256Sum != s {
		return nil, fmt.Errorf("httpx: SHA256 mismatch: got %s and expected %s", s, SHA256Sum)
	}
	return data, nil
}

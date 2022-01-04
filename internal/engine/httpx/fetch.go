package httpx

import (
	"context"
)

// FetchResource fetches the specified resource and returns it.
func (c Client) FetchResource(ctx context.Context, URLPath string) ([]byte, error) {
	request, err := c.NewRequest(ctx, "GET", URLPath, nil, nil)
	if err != nil {
		return nil, err
	}
	return c.Do(request)
}

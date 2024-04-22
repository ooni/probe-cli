package probeservices

//
// register.go - POST /api/v1/register
//

import (
	"context"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/randx"
)

// Register invokes the /api/v1/register API.
func (c *Client) Register(
	ctx context.Context, input *model.OOAPIRegisterRequest) (*model.OOAPIRegisterResponse, error) {
	// construct the URL to use
	URL, err := url.Parse(c.BaseURL)
	if err != nil {
		return nil, err
	}
	URL.Path = "/api/v1/register"

	// get the response
	return httpclientx.PostJSON[*model.OOAPIRegisterRequest, *model.OOAPIRegisterResponse](
		ctx, URL.String(), c.HTTPClient, input, c.Logger, c.UserAgent)
}

// MaybeRegister registers this client if not already registered
func (c Client) MaybeRegister(ctx context.Context, metadata model.OOAPIProbeMetadata) error {
	if !metadata.Valid() {
		return ErrInvalidMetadata
	}
	state := c.StateFile.Get()
	if state.Credentials() != nil {
		return nil // we're already good
	}
	c.RegisterCalls.Add(1)
	// TODO(bassosimone): here we should use a CSRNG
	// (https://github.com/ooni/probe/issues/1502)
	pwd := randx.Letters(64)
	req := &model.OOAPIRegisterRequest{
		OOAPIProbeMetadata: metadata,
		Password:           pwd,
	}
	var resp model.OOAPIRegisterResponse
	if err := c.APIClientTemplate.Build().PostJSON(
		ctx, "/api/v1/register", req, &resp); err != nil {
		return err
	}
	state.ClientID = resp.ClientID
	state.Password = pwd
	return c.StateFile.Set(state)
}

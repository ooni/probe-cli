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

	URL, err := url.Parse(c.BaseURL)
	if err != nil {
		return err
	}
	URL.Path = "/api/v1/register"

	resp, err := httpclientx.PostJSON[*model.OOAPIRegisterRequest, *model.OOAPIRegisterResponse](
		ctx, URL.String(), req, &httpclientx.Config{
			Client:    c.HTTPClient,
			Logger:    c.Logger,
			UserAgent: c.UserAgent,
		},
	)
	if err != nil {
		return err
	}

	state.ClientID = resp.ClientID
	state.Password = pwd
	return c.StateFile.Set(state)
}

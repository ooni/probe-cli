package probeservices

//
// register.go - POST /api/v1/register
//

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/randx"
	"github.com/ooni/probe-cli/v3/internal/urlx"
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

	URL, err := urlx.ResolveReference(c.BaseURL, "/api/v1/register", "")
	if err != nil {
		return err
	}

	resp, err := httpclientx.PostJSON[*model.OOAPIRegisterRequest, *model.OOAPIRegisterResponse](
		ctx,
		httpclientx.NewEndpoint(URL).WithHostOverride(c.Host),
		req,
		&httpclientx.Config{
			Client:    c.HTTPClient,
			Logger:    model.DiscardLogger,
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

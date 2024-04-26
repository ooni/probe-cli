package probeservices

import (
	"context"

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
	var resp model.OOAPIRegisterResponse
	if err := c.APIClientTemplate.Build().PostJSON(
		ctx, "/api/v1/register", req, &resp); err != nil {
		return err
	}
	state.ClientID = resp.ClientID
	state.Password = pwd
	return c.StateFile.Set(state)
}

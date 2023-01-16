package probeservices

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/model"
)

// MaybeLogin performs login if necessary
func (c Client) MaybeLogin(ctx context.Context) error {
	state := c.StateFile.Get()
	if state.Auth() != nil {
		return nil // we're already good
	}
	creds := state.Credentials()
	if creds == nil {
		return ErrNotRegistered
	}
	c.LoginCalls.Add(1)
	var auth model.OOAPILoginAuth
	if err := c.APIClientTemplate.Build().PostJSON(
		ctx, "/api/v1/login", *creds, &auth); err != nil {
		return err
	}
	state.Expire = auth.Expire
	state.Token = auth.Token
	return c.StateFile.Set(state)
}

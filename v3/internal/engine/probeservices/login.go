package probeservices

import (
	"context"
	"time"
)

// LoginCredentials contains the login credentials
type LoginCredentials struct {
	ClientID string `json:"username"`
	Password string `json:"password"`
}

// LoginAuth contains authentication info
type LoginAuth struct {
	Expire time.Time `json:"expire"`
	Token  string    `json:"token"`
}

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
	var auth LoginAuth
	if err := c.Client.PostJSON(ctx, "/api/v1/login", *creds, &auth); err != nil {
		return err
	}
	state.Expire = auth.Expire
	state.Token = auth.Token
	return c.StateFile.Set(state)
}

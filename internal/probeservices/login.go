package probeservices

import (
	"context"

	"github.com/ooni/probe-cli/v3/internal/httpclientx"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/urlx"
)

// MaybeLogin performs login if necessary
func (c Client) MaybeLogin(ctx context.Context, baseURL string) error {
	state := c.StateFile.Get()
	if state.Auth() != nil {
		return nil // we're already good
	}
	creds := state.Credentials()
	if creds == nil {
		return ErrNotRegistered
	}
	c.LoginCalls.Add(1)

	// construct the URL to use
	if baseURL == "" {
		baseURL = c.BaseURL // fallback to the client BaseURL if the passed url is empty
	}
	URL, err := urlx.ResolveReference(baseURL, "/api/v1/login", "")
	if err != nil {
		return err
	}

	auth, err := httpclientx.PostJSON[*model.OOAPILoginCredentials, *model.OOAPILoginAuth](
		ctx,
		httpclientx.NewEndpoint(URL).WithHostOverride(c.Host),
		creds,
		&httpclientx.Config{
			Client:    c.HTTPClient,
			Logger:    model.DiscardLogger,
			UserAgent: c.UserAgent,
		},
	)

	if err != nil {
		return err
	}

	state.Expire = auth.Expire
	state.Token = auth.Token
	return c.StateFile.Set(state)
}

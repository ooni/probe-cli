package backendclient

import (
	"context"
	"net/url"

	"github.com/ooni/probe-cli/v3/internal/httpapi"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/ooapi"
)

type Config struct {
	KVStore    model.KeyValueStore
	HTTPClient model.HTTPClient
	Logger     model.Logger
	UserAgent  string

	// optional fields
	BaseURL  *url.URL
	ProxyURL *url.URL
}

type Client struct {
	endpoint *httpapi.Endpoint
}

func New(config *Config) *Client {
	baseURL := "https://api.ooni.io/"
	if config.BaseURL != nil {
		baseURL = config.BaseURL.String()
	}
	endpoint := &httpapi.Endpoint{
		BaseURL:    baseURL,
		HTTPClient: config.HTTPClient,
		Host:       "",
		Logger:     config.Logger,
		UserAgent:  config.UserAgent,
	}
	backendClient := &Client{
		endpoint: endpoint,
	}
	return backendClient
}

func (c *Client) CheckIn(
	ctx context.Context, config *model.OOAPICheckInConfig) (*model.OOAPICheckInResult, error) {
	return httpapi.Call(ctx, ooapi.NewDescriptorCheckIn(config), c.endpoint)
}

func (c *Client) FetchPsiphonConfig(ctx context.Context) ([]byte, error) {
	panic("not implemented")
}

func (c *Client) FetchTorTargets(
	ctx context.Context, cc string) (result map[string]model.OOAPITorTarget, err error) {
	panic("not implemented")
}

func (c *Client) Submit(ctx context.Context, m *model.Measurement) error {
	req := &model.OOAPICollectorUpdateRequest{
		Format:  "json",
		Content: m,
	}
	descriptor := newSubmitDescriptor(req, m.ReportID)
	_, err := httpapi.Call(ctx, descriptor, c.endpoint)
	return err
}

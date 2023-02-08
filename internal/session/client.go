package session

import (
	"context"
	"sync"

	"github.com/ooni/probe-cli/v3/internal/geolocate"
	"github.com/ooni/probe-cli/v3/internal/model"
)

// Client is a client for a [Session]. You should use a [Client]
// when you are writing Go code that uses a [Session].
type Client struct {
	logger  model.Logger
	once    sync.Once
	session *Session
}

// NewClient creates a [Client] instance.
func NewClient(logger model.Logger) *Client {
	return &Client{
		logger:  logger,
		once:    sync.Once{},
		session: New(),
	}
}

// Bootstrap bootstraps a session.
func (c *Client) Bootstrap(ctx context.Context, req *BootstrapRequest) error {
	if err := c.session.Send(ctx, &Request{Bootstrap: req}); err != nil {
		return err
	}
	for {
		resp, err := c.session.Recv(ctx)
		if err != nil {
			return err
		}
		if resp.Log != nil {
			c.emitLog(resp.Log)
			continue
		}
		if resp.Ticker != nil {
			c.logger.Infof("bootstrap in progress (elapsed: %+v)", resp.Ticker.ElapsedTime)
			continue
		}
		if resp.Bootstrap != nil {
			return resp.Bootstrap.Error
		}
		c.logger.Warnf("unexpected event: %+v", resp)
	}
}

// Geolocate performs geolocation. You must run bootstrap first.
func (c *Client) Geolocate(ctx context.Context, req *GeolocateRequest) (*geolocate.Results, error) {
	if err := c.session.Send(ctx, &Request{Geolocate: req}); err != nil {
		return nil, err
	}
	for {
		resp, err := c.session.Recv(ctx)
		if err != nil {
			return nil, err
		}
		if resp.Log != nil {
			c.emitLog(resp.Log)
			continue
		}
		if resp.Ticker != nil {
			c.logger.Infof("geolocate in progress (elapsed: %+v)", resp.Ticker.ElapsedTime)
			continue
		}
		if resp.Geolocate != nil {
			return resp.Geolocate.Location, resp.Geolocate.Error
		}
		c.logger.Warnf("unexpected event: %+v", resp)
	}
}

// CheckIn calls the check-in API. You must run bootstrap first.
func (c *Client) CheckIn(ctx context.Context, req *CheckInRequest) (*model.OOAPICheckInResult, error) {
	if err := c.session.Send(ctx, &Request{CheckIn: req}); err != nil {
		return nil, err
	}
	for {
		resp, err := c.session.Recv(ctx)
		if err != nil {
			return nil, err
		}
		if resp.Log != nil {
			c.emitLog(resp.Log)
			continue
		}
		if resp.Ticker != nil {
			c.logger.Infof("check-in in progress (elapsed: %+v)", resp.Ticker.ElapsedTime)
			continue
		}
		if resp.CheckIn != nil {
			return resp.CheckIn.Result, resp.CheckIn.Error
		}
		c.logger.Warnf("unexpected event: %+v", resp)
	}
}

// WebConnectivity runs a single-URL Web Connectivity measurement.
func (c *Client) WebConnectivity(
	ctx context.Context, req *WebConnectivityRequest) (*model.Measurement, error) {
	if err := c.session.Send(ctx, &Request{WebConnectivity: req}); err != nil {
		return nil, err
	}
	for {
		resp, err := c.session.Recv(ctx)
		if err != nil {
			return nil, err
		}
		if resp.Log != nil {
			c.emitLog(resp.Log)
			continue
		}
		if resp.Ticker != nil {
			c.logger.Infof("webconnectivity (elapsed: %+v)", resp.Ticker.ElapsedTime)
			continue
		}
		if resp.WebConnectivity != nil {
			return resp.WebConnectivity.Measurement, resp.WebConnectivity.Error
		}
		c.logger.Warnf("unexpected event: %+v", resp)
	}
}

// Submit submits a measurement.
func (c *Client) Submit(ctx context.Context, measurement *model.Measurement) error {
	if err := c.session.Send(ctx, &Request{Submit: measurement}); err != nil {
		return err
	}
	for {
		resp, err := c.session.Recv(ctx)
		if err != nil {
			return err
		}
		if resp.Log != nil {
			c.emitLog(resp.Log)
			continue
		}
		if resp.Ticker != nil {
			c.logger.Infof("webconnectivity (elapsed: %+v)", resp.Ticker.ElapsedTime)
			continue
		}
		if resp.Submit != nil {
			return resp.Submit.Error
		}
		c.logger.Warnf("unexpected event: %+v", resp)
	}
}

// Close releases the resources used by a [Client].
func (c *Client) Close() (err error) {
	c.once.Do(func() {
		err = c.session.Close()
	})
	return
}

// emitLog emits a log event.
func (c *Client) emitLog(ev *LogEvent) {
	switch ev.Level {
	case "DEBUG":
		c.logger.Debug(ev.Message)
	case "WARNING":
		c.logger.Warn(ev.Message)
	default:
		c.logger.Info(ev.Message)
	}
}

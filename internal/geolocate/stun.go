package geolocate

import (
	"context"
	"net"
	"net/http"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/pion/stun"
)

type stunClient interface {
	Close() error
	Start(m *stun.Message, h stun.Handler) error
}

type stunConfig struct {
	Dialer    model.Dialer // optional
	Endpoint  string
	Logger    model.Logger
	NewClient func(conn net.Conn) (stunClient, error) // optional
	Resolver  model.Resolver
}

func stunNewClient(conn net.Conn) (stunClient, error) {
	return stun.NewClient(conn)
}

func stunIPLookup(ctx context.Context, config stunConfig) (string, error) {
	config.Logger.Debugf("STUNIPLookup: start using %s", config.Endpoint)
	ip, err := func() (string, error) {
		dialer := config.Dialer
		if dialer == nil {
			dialer = netxlite.NewDialerWithResolver(config.Logger, config.Resolver)
		}
		conn, err := dialer.DialContext(ctx, "udp", config.Endpoint)
		if err != nil {
			return model.DefaultProbeIP, err
		}
		newClient := config.NewClient
		if newClient == nil {
			newClient = stunNewClient
		}
		clnt, err := newClient(conn)
		if err != nil {
			conn.Close()
			return model.DefaultProbeIP, err
		}
		defer clnt.Close()
		message := stun.MustBuild(stun.TransactionID, stun.BindingRequest)
		errch, ipch := make(chan error, 1), make(chan string, 1)
		err = clnt.Start(message, func(ev stun.Event) {
			if ev.Error != nil {
				errch <- ev.Error
				return
			}
			var xorAddr stun.XORMappedAddress
			if err := xorAddr.GetFrom(ev.Message); err != nil {
				errch <- err
				return
			}
			ipch <- xorAddr.IP.String()
		})
		if err != nil {
			return model.DefaultProbeIP, err
		}
		select {
		case err := <-errch:
			return model.DefaultProbeIP, err
		case ip := <-ipch:
			return ip, nil
		case <-ctx.Done():
			return model.DefaultProbeIP, ctx.Err()
		}
	}()
	if err != nil {
		config.Logger.Debugf("STUNIPLookup: failure using %s: %+v", config.Endpoint, err)
		return model.DefaultProbeIP, err
	}
	return ip, nil
}

func stunEkigaIPLookup(
	ctx context.Context,
	httpClient *http.Client,
	logger model.Logger,
	userAgent string,
	resolver model.Resolver,
) (string, error) {
	return stunIPLookup(ctx, stunConfig{
		Endpoint: "stun.ekiga.net:3478",
		Logger:   logger,
		Resolver: resolver,
	})
}

func stunGoogleIPLookup(
	ctx context.Context,
	httpClient *http.Client,
	logger model.Logger,
	userAgent string,
	resolver model.Resolver,
) (string, error) {
	return stunIPLookup(ctx, stunConfig{
		Endpoint: "stun.l.google.com:19302",
		Logger:   logger,
		Resolver: resolver,
	})
}

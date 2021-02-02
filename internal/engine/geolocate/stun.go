package geolocate

import (
	"context"
	"net/http"

	"github.com/pion/stun"
)

type stunClient interface {
	Close() error
	Start(m *stun.Message, h stun.Handler) error
}

type stunConfig struct {
	Dial     func(network string, address string) (stunClient, error)
	Endpoint string
	Logger   Logger
}

func stunDialer(network string, address string) (stunClient, error) {
	return stun.Dial(network, address)
}

func stunIPLookup(ctx context.Context, config stunConfig) (string, error) {
	config.Logger.Debugf("STUNIPLookup: start using %s", config.Endpoint)
	ip, err := func() (string, error) {
		dial := config.Dial
		if dial == nil {
			dial = stunDialer
		}
		clnt, err := dial("udp", config.Endpoint)
		if err != nil {
			return DefaultProbeIP, err
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
			return DefaultProbeIP, err
		}
		select {
		case err := <-errch:
			return DefaultProbeIP, err
		case ip := <-ipch:
			return ip, nil
		case <-ctx.Done():
			return DefaultProbeIP, ctx.Err()
		}
	}()
	if err != nil {
		config.Logger.Debugf("STUNIPLookup: failure using %s: %+v", config.Endpoint, err)
		return DefaultProbeIP, err
	}
	return ip, nil
}

func stunEkigaIPLookup(
	ctx context.Context,
	httpClient *http.Client,
	logger Logger,
	userAgent string,
) (string, error) {
	return stunIPLookup(ctx, stunConfig{
		Endpoint: "stun.ekiga.net:3478",
		Logger:   logger,
	})
}

func stunGoogleIPLookup(
	ctx context.Context,
	httpClient *http.Client,
	logger Logger,
	userAgent string,
) (string, error) {
	return stunIPLookup(ctx, stunConfig{
		Endpoint: "stun.l.google.com:19302",
		Logger:   logger,
	})
}

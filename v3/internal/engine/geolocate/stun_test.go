package geolocate

import (
	"context"
	"errors"
	"net"
	"net/http"
	"testing"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/internal/httpheader"
	"github.com/pion/stun"
)

func TestSTUNIPLookupCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // stop immediately
	ip, err := stunIPLookup(ctx, stunConfig{
		Endpoint: "stun.ekiga.net:3478",
		Logger:   log.Log,
	})
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if ip != DefaultProbeIP {
		t.Fatalf("not the IP address we expected: %+v", ip)
	}
}

func TestSTUNIPLookupDialFailure(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	ip, err := stunIPLookup(ctx, stunConfig{
		Dial: func(network, address string) (stunClient, error) {
			return nil, expected
		},
		Endpoint: "stun.ekiga.net:3478",
		Logger:   log.Log,
	})
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if ip != DefaultProbeIP {
		t.Fatalf("not the IP address we expected: %+v", ip)
	}
}

type MockableSTUNClient struct {
	StartErr error
	Event    stun.Event
}

func (c MockableSTUNClient) Close() error {
	return nil
}

func (c MockableSTUNClient) Start(m *stun.Message, h stun.Handler) error {
	if c.StartErr != nil {
		return c.StartErr
	}
	go func() {
		<-time.After(100 * time.Millisecond)
		h(c.Event)
	}()
	return nil
}

func TestSTUNIPLookupStartReturnsError(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	ip, err := stunIPLookup(ctx, stunConfig{
		Dial: func(network, address string) (stunClient, error) {
			return MockableSTUNClient{StartErr: expected}, nil
		},
		Endpoint: "stun.ekiga.net:3478",
		Logger:   log.Log,
	})
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if ip != DefaultProbeIP {
		t.Fatalf("not the IP address we expected: %+v", ip)
	}
}

func TestSTUNIPLookupStunEventContainsError(t *testing.T) {
	expected := errors.New("mocked error")
	ctx := context.Background()
	ip, err := stunIPLookup(ctx, stunConfig{
		Dial: func(network, address string) (stunClient, error) {
			return MockableSTUNClient{Event: stun.Event{
				Error: expected,
			}}, nil
		},
		Endpoint: "stun.ekiga.net:3478",
		Logger:   log.Log,
	})
	if !errors.Is(err, expected) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if ip != DefaultProbeIP {
		t.Fatalf("not the IP address we expected: %+v", ip)
	}
}

func TestSTUNIPLookupCannotDecodeMessage(t *testing.T) {
	ctx := context.Background()
	ip, err := stunIPLookup(ctx, stunConfig{
		Dial: func(network, address string) (stunClient, error) {
			return MockableSTUNClient{Event: stun.Event{
				Message: &stun.Message{},
			}}, nil
		},
		Endpoint: "stun.ekiga.net:3478",
		Logger:   log.Log,
	})
	if !errors.Is(err, stun.ErrAttributeNotFound) {
		t.Fatalf("not the error we expected: %+v", err)
	}
	if ip != DefaultProbeIP {
		t.Fatalf("not the IP address we expected: %+v", ip)
	}
}

func TestIPLookupWorksUsingSTUNEkiga(t *testing.T) {
	ip, err := stunEkigaIPLookup(
		context.Background(),
		http.DefaultClient,
		log.Log,
		httpheader.UserAgent(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if net.ParseIP(ip) == nil {
		t.Fatalf("not an IP address: '%s'", ip)
	}
}

func TestIPLookupWorksUsingSTUNGoogle(t *testing.T) {
	ip, err := stunGoogleIPLookup(
		context.Background(),
		http.DefaultClient,
		log.Log,
		httpheader.UserAgent(),
	)
	if err != nil {
		t.Fatal(err)
	}
	if net.ParseIP(ip) == nil {
		t.Fatalf("not an IP address: '%s'", ip)
	}
}

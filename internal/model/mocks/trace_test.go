package mocks

import (
	"crypto/tls"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

func TestTrace(t *testing.T) {
	t.Run("TimeNow", func(t *testing.T) {
		now := time.Now()
		tx := &Trace{
			MockTimeNow: func() time.Time {
				return now
			},
		}
		if !tx.TimeNow().Equal(now) {
			t.Fatal("not working as intended")
		}
	})

	t.Run("OnDNSRoundTripForLookupHost", func(t *testing.T) {
		var called bool
		tx := &Trace{
			MockOnDNSRoundTripForLookupHost: func(started time.Time, reso model.Resolver, query model.DNSQuery,
				response model.DNSResponse, addrs []string, err error, finished time.Time) {
				called = true
			},
		}
		tx.OnDNSRoundTripForLookupHost(
			time.Now(),
			&Resolver{},
			&DNSQuery{},
			&DNSResponse{},
			[]string{},
			nil,
			time.Now(),
		)
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("OnConnectDone", func(t *testing.T) {
		var called bool
		tx := &Trace{
			MockOnConnectDone: func(started time.Time, network, domain, remoteAddr string, err error, finished time.Time) {
				called = true
			},
		}
		tx.OnConnectDone(
			time.Now(),
			"tcp",
			"dns.google",
			"8.8.8.8:443",
			nil,
			time.Now(),
		)
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("OnTLSHandshakeStart", func(t *testing.T) {
		var called bool
		tx := &Trace{
			MockOnTLSHandshakeStart: func(now time.Time, remoteAddr string, config *tls.Config) {
				called = true
			},
		}
		tx.OnTLSHandshakeStart(time.Now(), "8.8.8.8:443", &tls.Config{})
		if !called {
			t.Fatal("not called")
		}
	})

	t.Run("OnTLSHandshakeDone", func(t *testing.T) {
		var called bool
		tx := &Trace{
			MockOnTLSHandshakeDone: func(started time.Time, remoteAddr string, config *tls.Config, state tls.ConnectionState, err error, finished time.Time) {
				called = true
			},
		}
		tx.OnTLSHandshakeDone(
			time.Now(),
			"8.8.8.8:443",
			&tls.Config{},
			tls.ConnectionState{},
			nil,
			time.Now(),
		)
		if !called {
			t.Fatal("not called")
		}
	})
}

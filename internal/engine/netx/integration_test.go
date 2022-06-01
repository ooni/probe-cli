package netx_test

import (
	"context"
	"errors"
	"net/http"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

func TestSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	log.SetLevel(log.DebugLevel)
	counter := bytecounter.New()
	config := netx.Config{
		BogonIsError:        true,
		ByteCounter:         counter,
		CacheResolutions:    true,
		ContextByteCounting: true,
		DialSaver:           &tracex.Saver{},
		HTTPSaver:           &tracex.Saver{},
		Logger:              log.Log,
		ReadWriteSaver:      &tracex.Saver{},
		ResolveSaver:        &tracex.Saver{},
		TLSSaver:            &tracex.Saver{},
	}
	txp := netx.NewHTTPTransport(config)
	client := &http.Client{Transport: txp}
	resp, err := client.Get("https://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	if _, err = netxlite.ReadAllContext(context.Background(), resp.Body); err != nil {
		t.Fatal(err)
	}
	if err = resp.Body.Close(); err != nil {
		t.Fatal(err)
	}
	if counter.Sent.Load() <= 0 {
		t.Fatal("no bytes sent?!")
	}
	if counter.Received.Load() <= 0 {
		t.Fatal("no bytes received?!")
	}
	if ev := config.DialSaver.Read(); len(ev) <= 0 {
		t.Fatal("no dial events?!")
	}
	if ev := config.HTTPSaver.Read(); len(ev) <= 0 {
		t.Fatal("no HTTP events?!")
	}
	if ev := config.ReadWriteSaver.Read(); len(ev) <= 0 {
		t.Fatal("no R/W events?!")
	}
	if ev := config.ResolveSaver.Read(); len(ev) <= 0 {
		t.Fatal("no resolver events?!")
	}
	if ev := config.TLSSaver.Read(); len(ev) <= 0 {
		t.Fatal("no TLS events?!")
	}
}

func TestBogonResolutionNotBroken(t *testing.T) {
	saver := new(tracex.Saver)
	r := netx.NewResolver(netx.Config{
		BogonIsError: true,
		DNSCache: map[string][]string{
			"www.google.com": {"127.0.0.1"},
		},
		ResolveSaver: saver,
		Logger:       log.Log,
	})
	addrs, err := r.LookupHost(context.Background(), "www.google.com")
	if !errors.Is(err, netxlite.ErrDNSBogon) {
		t.Fatal("not the error we expected")
	}
	if err.Error() != netxlite.FailureDNSBogonError {
		t.Fatal("error not correctly wrapped")
	}
	if len(addrs) > 0 {
		t.Fatal("expected no addresses here")
	}
}

package netx

import (
	"context"
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/tracex"
)

func TestNewResolverBogonResolutionNotBroken(t *testing.T) {
	saver := new(tracex.Saver)
	r := NewResolver(Config{
		BogonIsError: true,
		DNSCache: map[string][]string{
			"www.google.com": {"127.0.0.1"},
		},
		Saver:  saver,
		Logger: log.Log,
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

package netemx

import (
	"context"
	"errors"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

// Make sure we can emulate Android's getaddrinfo behavior.
func TestEmulateAndroidGetaddrinfo(t *testing.T) {
	env := MustNewScenario(InternetScenario)
	defer env.Close()

	env.EmulateAndroidGetaddrinfo(true)
	defer env.EmulateAndroidGetaddrinfo(false)

	env.Do(func() {
		netx := &netxlite.Netx{}
		reso := netx.NewStdlibResolver(log.Log)
		addrs, err := reso.LookupHost(context.Background(), "www.nonexistent.xyz")
		if !errors.Is(err, netxlite.ErrAndroidDNSCacheNoData) {
			t.Fatal("unexpected error")
		}
		if len(addrs) != 0 {
			t.Fatal("expected zero-length addresses")
		}
	})
}

package measurex

import (
	"context"
	"testing"

	"github.com/miekg/dns"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestDNSXModifiesStdlibTransportName(t *testing.T) {
	// See https://github.com/ooni/spec/pull/257 for more information.
	child := netxlite.NewDNSOverGetaddrinfoTransport()
	mx := NewMeasurerWithDefaultSettings()
	dbout := &MeasurementDB{}
	txp := mx.WrapDNSXRoundTripper(dbout, child)
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // we want to fail immediately
	query := &mocks.DNSQuery{
		MockDomain: func() string {
			return "dns.google"
		},
		MockType: func() uint16 {
			return dns.TypeANY
		},
		MockBytes: func() ([]byte, error) {
			return []byte{}, nil
		},
		MockID: func() uint16 {
			return 1453
		},
	}
	_, _ = txp.RoundTrip(ctx, query)
	measurement := dbout.AsMeasurement()
	var good int
	for _, rtinfo := range measurement.DNSRoundTrip {
		network := rtinfo.Network
		if network != netxlite.StdlibResolverSystem {
			t.Fatal("unexpected network", network)
		}
		good++
	}
	if good < 1 {
		t.Fatal("no good entry seen")
	}
}

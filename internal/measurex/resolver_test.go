package measurex

import (
	"context"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestResolverModifiesStdlibResolverName(t *testing.T) {
	// See https://github.com/ooni/spec/pull/257 for more information.

	t.Run("for LookupHost", func(t *testing.T) {
		child := netxlite.NewStdlibResolver(model.DiscardLogger)
		mx := NewMeasurerWithDefaultSettings()
		dbout := &MeasurementDB{}
		txp := mx.WrapResolver(dbout, child)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // we want to fail immediately
		_, _ = txp.LookupHost(ctx, "dns.google")
		measurement := dbout.AsMeasurement()
		var good int
		for _, rtinfo := range measurement.LookupHost {
			network := rtinfo.Network
			if network != netxlite.StdlibResolverSystem {
				t.Fatal("unexpected network", network)
			}
			good++
		}
		if good < 1 {
			t.Fatal("no good entry seen")
		}
	})

	t.Run("for LookupHTTPS", func(t *testing.T) {
		child := netxlite.NewStdlibResolver(model.DiscardLogger)
		mx := NewMeasurerWithDefaultSettings()
		dbout := &MeasurementDB{}
		txp := mx.WrapResolver(dbout, child)
		ctx, cancel := context.WithCancel(context.Background())
		cancel() // we want to fail immediately
		_, _ = txp.LookupHTTPS(ctx, "dns.google")
		measurement := dbout.AsMeasurement()
		var good int
		for _, rtinfo := range measurement.LookupHTTPSSvc {
			network := rtinfo.Network
			if network != netxlite.StdlibResolverSystem {
				t.Fatal("unexpected network", network)
			}
			good++
		}
		if good < 1 {
			t.Fatal("no good entry seen")
		}
	})

}

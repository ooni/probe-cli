package dialer

import (
	"net"
	"net/url"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestNewCreatesTheExpectedChain(t *testing.T) {
	saver := &trace.Saver{}
	dlr := New(&Config{
		ContextByteCounting: true,
		DialSaver:           saver,
		Logger:              log.Log,
		ProxyURL:            &url.URL{},
		ReadWriteSaver:      saver,
	}, &net.Resolver{})
	shd, ok := dlr.(*shapingDialer)
	if !ok {
		t.Fatal("not a shapingDialer")
	}
	bcd, ok := shd.Dialer.(*byteCounterDialer)
	if !ok {
		t.Fatal("not a byteCounterDialer")
	}
	pd, ok := bcd.Dialer.(*proxyDialer)
	if !ok {
		t.Fatal("not a proxyDialer")
	}
	dnsd, ok := pd.Dialer.(*netxlite.DialerResolver)
	if !ok {
		t.Fatal("not a dnsDialer")
	}
	dad, ok := dnsd.Dialer.(*netxlite.DialerLegacyAdapter)
	if !ok {
		t.Fatal("invalid type")
	}
	scd, ok := dad.DialerLegacy.(*saverConnDialer)
	if !ok {
		t.Fatal("not a saverConnDialer")
	}
	sd, ok := scd.Dialer.(*saverDialer)
	if !ok {
		t.Fatal("not a saverDialer")
	}
	ld, ok := sd.Dialer.(*netxlite.DialerLogger)
	if !ok {
		t.Fatal("not a loggingDialer")
	}
	dad, ok = ld.Dialer.(*netxlite.DialerLegacyAdapter)
	if !ok {
		t.Fatal("invalid type")
	}
	ewd, ok := dad.DialerLegacy.(*errorsx.ErrorWrapperDialer)
	if !ok {
		t.Fatal("not an errorWrappingDialer")
	}
	_, ok = ewd.Dialer.(*netxlite.DialerSystem)
	if !ok {
		t.Fatal("not a DialerSystem")
	}
}

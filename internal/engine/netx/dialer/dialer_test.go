package dialer

import (
	"net"
	"net/url"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
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
	dnsd, ok := pd.Dialer.(*dnsDialer)
	if !ok {
		t.Fatal("not a dnsDialer")
	}
	scd, ok := dnsd.Dialer.(*saverConnDialer)
	if !ok {
		t.Fatal("not a saverConnDialer")
	}
	sd, ok := scd.Dialer.(*saverDialer)
	if !ok {
		t.Fatal("not a saverDialer")
	}
	ld, ok := sd.Dialer.(*loggingDialer)
	if !ok {
		t.Fatal("not a loggingDialer")
	}
	ewd, ok := ld.Dialer.(*errorWrapperDialer)
	if !ok {
		t.Fatal("not an errorWrappingDialer")
	}
	_, ok = ewd.Dialer.(*net.Dialer)
	if !ok {
		t.Fatal("not a net.Dialer")
	}
}

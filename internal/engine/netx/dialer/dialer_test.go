package dialer

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
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
	}, netxlite.DefaultResolver)
	bcd, ok := dlr.(*bytecounter.ContextAwareDialer)
	if !ok {
		t.Fatal("not a byteCounterDialer")
	}
	pd, ok := bcd.Dialer.(*netxlite.MaybeProxyDialer)
	if !ok {
		t.Fatal("not a proxyDialer")
	}
	dnsd, ok := pd.Dialer.(*netxlite.DialerResolver)
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
	ld, ok := sd.Dialer.(*netxlite.DialerLogger)
	if !ok {
		t.Fatal("not a loggingDialer")
	}
	ewd, ok := ld.Dialer.(*netxlite.ErrorWrapperDialer)
	if !ok {
		t.Fatal("not an errorWrappingDialer")
	}
	_, ok = ewd.Dialer.(*netxlite.DialerSystem)
	if !ok {
		t.Fatal("not a DialerSystem")
	}
}

func TestDialerNewSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	log.SetLevel(log.DebugLevel)
	d := New(&Config{Logger: log.Log}, netxlite.DefaultResolver)
	txp := &http.Transport{DialContext: d.DialContext}
	client := &http.Client{Transport: txp}
	resp, err := client.Get("http://www.google.com")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
}

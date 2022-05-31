package dialer

import (
	"net/http"
	"net/url"
	"testing"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/bytecounter"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tracex"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestNewCreatesTheExpectedChain(t *testing.T) {
	saver := &tracex.Saver{}
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
	_, ok = bcd.Dialer.(*netxlite.MaybeProxyDialer)
	if !ok {
		t.Fatal("not a proxyDialer")
	}
	// We can safely stop here: the rest is tested by
	// the internal/netxlite package
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

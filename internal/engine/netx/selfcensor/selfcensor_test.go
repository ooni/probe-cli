package selfcensor_test

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/selfcensor"
)

// TestDisabled MUST be the first test in this file.
func TestDisabled(t *testing.T) {
	if selfcensor.Enabled() != false {
		t.Fatal("self censorship should be disabled by default")
	}
	if selfcensor.Attempts() != 0 {
		t.Fatal("we expect no self censorship attempts at the beginning")
	}
	t.Run("the system resolver does not trigger selfcensor events", func(t *testing.T) {
		addrs, err := selfcensor.SystemResolver{}.LookupHost(
			context.Background(), "dns.google",
		)
		if err != nil {
			t.Fatal(err)
		}
		if addrs == nil {
			t.Fatal("expected non-nil addrs here")
		}
		if selfcensor.Attempts() != 0 {
			t.Fatal("we expect no self censorship attempts by default")
		}
	})
	t.Run("the system dialer does not trigger selfcensor events", func(t *testing.T) {
		conn, err := selfcensor.SystemDialer{}.DialContext(
			context.Background(), "tcp", "8.8.8.8:443",
		)
		if err != nil {
			t.Fatal(err)
		}
		if conn == nil {
			t.Fatal("expected non-nil conn here")
		}
		conn.Close()
		if selfcensor.Attempts() != 0 {
			t.Fatal("we expect no self censorship attempts by default")
		}
	})
}

// TestDisabled MUST be the second test in this file.
func TestEnableInvalidJSON(t *testing.T) {
	if selfcensor.Enabled() != false {
		t.Fatal("we need to start with self censorship not enabled")
	}
	err := selfcensor.Enable("{")
	if err == nil || !strings.HasSuffix(err.Error(), "unexpected end of JSON input") {
		t.Fatal("not the error we expectd")
	}
	if selfcensor.Enabled() != false {
		t.Fatal("we expected self censorship to still be not enabled")
	}
}

// TestMaybeEnableWorksAsIntended MUST be the second test in this file.
func TestMaybeEnableWorksAsIntended(t *testing.T) {
	if selfcensor.Enabled() != false {
		t.Fatal("we need to start with self censorship not enabled")
	}
	err := selfcensor.MaybeEnable("")
	if err != nil {
		t.Fatal(err)
	}
	if selfcensor.Enabled() != false {
		t.Fatal("we expected self censorship to still be not enabled")
	}
}

func TestResolveCauseNXDOMAIN(t *testing.T) {
	err := selfcensor.MaybeEnable(`{"PoisonSystemDNS":{"dns.google":["NXDOMAIN"]}}`)
	if err != nil {
		t.Fatal(err)
	}
	if selfcensor.Enabled() != true {
		t.Fatal("we expected self censorship to be enabled now")
	}
	addrs, err := selfcensor.SystemResolver{}.LookupHost(
		context.Background(), "dns.google",
	)
	if err == nil || !strings.HasSuffix(err.Error(), "no such host") {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected nil addrs here")
	}
}

func TestResolveCauseTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := selfcensor.MaybeEnable(`{"PoisonSystemDNS":{"dns.google":["TIMEOUT"]}}`)
	if err != nil {
		t.Fatal(err)
	}
	if selfcensor.Enabled() != true {
		t.Fatal("we expected self censorship to be enabled now")
	}
	addrs, err := selfcensor.SystemResolver{}.LookupHost(ctx, "dns.google")
	if err == nil || err.Error() != "i/o timeout" {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected nil addrs here")
	}
}

func TestResolveCauseBogon(t *testing.T) {
	err := selfcensor.MaybeEnable(`{"PoisonSystemDNS":{"dns.google":["10.0.0.7"]}}`)
	if err != nil {
		t.Fatal(err)
	}
	if selfcensor.Enabled() != true {
		t.Fatal("we expected self censorship to be enabled now")
	}
	addrs, err := selfcensor.SystemResolver{}.LookupHost(
		context.Background(), "dns.google")
	if err != nil {
		t.Fatal(err)
	}
	if len(addrs) != 1 || addrs[0] != "10.0.0.7" {
		t.Fatal("not the addrs we expected")
	}
}

func TestResolveCheckNetworkAndAddress(t *testing.T) {
	err := selfcensor.MaybeEnable(`{"PoisonSystemDNS":{"dns.google":["10.0.0.7"]}}`)
	if err != nil {
		t.Fatal(err)
	}
	if selfcensor.Enabled() != true {
		t.Fatal("we expected self censorship to be enabled now")
	}
	reso := selfcensor.SystemResolver{}
	if reso.Network() != "system" {
		t.Fatal("invalid Network")
	}
	if reso.Address() != "" {
		t.Fatal("invalid Address")
	}
}

func TestDialHandlesErrorsWithBlockedFingerprints(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	cancel() // so we should fail immediately!
	err := selfcensor.MaybeEnable(`{"BlockedFingerprints":{"dns.google":"TIMEOUT"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if selfcensor.Enabled() != true {
		t.Fatal("we expected self censorship to be enabled now")
	}
	addrs, err := selfcensor.SystemDialer{}.DialContext(ctx, "tcp", "8.8.8.8:443")
	if err == nil || !strings.HasSuffix(err.Error(), "operation was canceled") {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected nil addrs here")
	}
}

func TestDialCauseTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()
	err := selfcensor.MaybeEnable(`{"BlockedEndpoints":{"8.8.8.8:443":"TIMEOUT"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if selfcensor.Enabled() != true {
		t.Fatal("we expected self censorship to be enabled now")
	}
	addrs, err := selfcensor.SystemDialer{}.DialContext(ctx, "tcp", "8.8.8.8:443")
	if err == nil || err.Error() != "i/o timeout" {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected nil addrs here")
	}
}

func TestDialCauseConnectionRefused(t *testing.T) {
	err := selfcensor.MaybeEnable(`{"BlockedEndpoints":{"8.8.8.8:443":"REJECT"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if selfcensor.Enabled() != true {
		t.Fatal("we expected self censorship to be enabled now")
	}
	addrs, err := selfcensor.SystemDialer{}.DialContext(
		context.Background(), "tcp", "8.8.8.8:443")
	if err == nil || !strings.HasSuffix(err.Error(), "connection refused") {
		t.Fatal("not the error we expected")
	}
	if addrs != nil {
		t.Fatal("expected nil addrs here")
	}
}

func TestBlockedFingerprintsTimeout(t *testing.T) {
	err := selfcensor.MaybeEnable(`{"BlockedFingerprints":{"dns.google":"TIMEOUT"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if selfcensor.Enabled() != true {
		t.Fatal("we expected self censorship to be enabled now")
	}
	tlsDialer := netx.NewTLSDialer(netx.Config{
		Dialer: selfcensor.SystemDialer{},
	})
	conn, err := tlsDialer.DialTLSContext(
		context.Background(), "tcp", "dns.google:443")
	if err == nil || err.Error() != "generic_timeout_error" {
		t.Fatal("not the error expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

func TestBlockedFingerprintsNoMatch(t *testing.T) {
	err := selfcensor.MaybeEnable(`{"BlockedFingerprints":{"ooni.io":"TIMEOUT"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if selfcensor.Enabled() != true {
		t.Fatal("we expected self censorship to be enabled now")
	}
	tlsDialer := netx.NewTLSDialer(netx.Config{
		Dialer: selfcensor.SystemDialer{},
	})
	conn, err := tlsDialer.DialTLSContext(
		context.Background(), "tcp", "dns.google:443")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil conn here")
	}
	conn.Close()
}

func TestBlockedFingerprintsConnectionReset(t *testing.T) {
	err := selfcensor.MaybeEnable(`{"BlockedFingerprints":{"dns.google":"RST"}}`)
	if err != nil {
		t.Fatal(err)
	}
	if selfcensor.Enabled() != true {
		t.Fatal("we expected self censorship to be enabled now")
	}
	tlsDialer := netx.NewTLSDialer(netx.Config{
		Dialer: selfcensor.SystemDialer{},
	})
	conn, err := tlsDialer.DialTLSContext(
		context.Background(), "tcp", "dns.google:443")
	if err == nil || err.Error() != "connection_reset" {
		t.Fatal("not the error we expected")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
}

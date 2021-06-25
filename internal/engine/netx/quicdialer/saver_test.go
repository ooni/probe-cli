package quicdialer_test

import (
	"context"
	"crypto/tls"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/quicdialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

type MockDialer struct {
	Dialer quicdialer.ContextDialer
	Sess   quic.EarlySession
	Err    error
}

func (d MockDialer) DialContext(ctx context.Context, network, host string,
	tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlySession, error) {
	if d.Dialer != nil {
		return d.Dialer.DialContext(ctx, network, host, tlsCfg, cfg)
	}
	return d.Sess, d.Err
}

func TestHandshakeSaverSuccess(t *testing.T) {
	nextprotos := []string{"h3"}
	servername := "www.google.com"
	tlsConf := &tls.Config{
		NextProtos: nextprotos,
		ServerName: servername,
	}
	saver := &trace.Saver{}
	dlr := quicdialer.HandshakeSaver{
		Dialer: &netxlite.QUICDialerQUICGo{
			QUICListener: &netxlite.QUICListenerStdlib{},
		},
		Saver: saver,
	}
	sess, err := dlr.DialContext(context.Background(), "udp",
		"216.58.212.164:443", tlsConf, &quic.Config{})
	if err != nil {
		t.Fatal("unexpected error", err)
	}
	if sess == nil {
		t.Fatal("unexpected nil sess")
	}
	ev := saver.Read()
	if len(ev) != 2 {
		t.Fatal("unexpected number of events")
	}
	if ev[0].Name != "quic_handshake_start" {
		t.Fatal("unexpected Name")
	}
	if ev[0].TLSServerName != "www.google.com" {
		t.Fatal("unexpected TLSServerName")
	}
	if !reflect.DeepEqual(ev[0].TLSNextProtos, nextprotos) {
		t.Fatal("unexpected TLSNextProtos")
	}
	if ev[0].Time.After(time.Now()) {
		t.Fatal("unexpected Time")
	}
	if ev[1].Duration <= 0 {
		t.Fatal("unexpected Duration")
	}
	if ev[1].Err != nil {
		t.Fatal("unexpected Err", ev[1].Err)
	}
	if ev[1].Name != "quic_handshake_done" {
		t.Fatal("unexpected Name")
	}
	if !reflect.DeepEqual(ev[1].TLSNextProtos, nextprotos) {
		t.Fatal("unexpected TLSNextProtos")
	}
	if ev[1].TLSServerName != "www.google.com" {
		t.Fatal("unexpected TLSServerName")
	}
	if ev[1].Time.Before(ev[0].Time) {
		t.Fatal("unexpected Time")
	}
}

func TestHandshakeSaverHostNameError(t *testing.T) {
	nextprotos := []string{"h3"}
	servername := "example.com"
	tlsConf := &tls.Config{
		NextProtos: nextprotos,
		ServerName: servername,
	}
	saver := &trace.Saver{}
	dlr := quicdialer.HandshakeSaver{
		Dialer: &netxlite.QUICDialerQUICGo{
			QUICListener: &netxlite.QUICListenerStdlib{},
		},
		Saver: saver,
	}
	sess, err := dlr.DialContext(context.Background(), "udp",
		"216.58.212.164:443", tlsConf, &quic.Config{})
	if err == nil {
		t.Fatal("expected an error here")
	}
	if sess != nil {
		t.Fatal("expected nil sess here")
	}
	for _, ev := range saver.Read() {
		if ev.Name != "quic_handshake_done" {
			continue
		}
		if ev.NoTLSVerify == true {
			t.Fatal("expected NoTLSVerify to be false")
		}
		if !strings.Contains(ev.Err.Error(),
			"certificate is valid for www.google.com, not "+servername) {
			t.Fatal("unexpected error", ev.Err)
		}
	}
}

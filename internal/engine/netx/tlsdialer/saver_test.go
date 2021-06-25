package tlsdialer_test

import (
	"context"
	"crypto/tls"
	"net"
	"reflect"
	"testing"
	"time"

	"github.com/ooni/probe-cli/v3/internal/engine/netx/dialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/errorx"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/tlsdialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestSaverTLSHandshakerSuccessWithReadWrite(t *testing.T) {
	// This is the most common use case for collecting reads, writes
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	nextprotos := []string{"h2"}
	saver := &trace.Saver{}
	tlsdlr := &netxlite.TLSDialer{
		Config: &tls.Config{NextProtos: nextprotos},
		Dialer: dialer.New(&dialer.Config{ReadWriteSaver: saver}, &net.Resolver{}),
		TLSHandshaker: tlsdialer.SaverTLSHandshaker{
			TLSHandshaker: &netxlite.TLSHandshakerStdlib{},
			Saver:         saver,
		},
	}
	// Implementation note: we don't close the connection here because it is
	// very handy to have the last event being the end of the handshake
	_, err := tlsdlr.DialTLSContext(context.Background(), "tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	ev := saver.Read()
	if len(ev) < 4 {
		// it's a bit tricky to be sure about the right number of
		// events because network conditions may influence that
		t.Fatal("unexpected number of events")
	}
	if ev[0].Name != "tls_handshake_start" {
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
	last := len(ev) - 1
	for idx := 1; idx < last; idx++ {
		if ev[idx].Data == nil {
			t.Fatal("unexpected Data")
		}
		if ev[idx].Duration <= 0 {
			t.Fatal("unexpected Duration")
		}
		if ev[idx].Err != nil {
			t.Fatal("unexpected Err")
		}
		if ev[idx].NumBytes <= 0 {
			t.Fatal("unexpected NumBytes")
		}
		switch ev[idx].Name {
		case errorx.ReadOperation, errorx.WriteOperation:
		default:
			t.Fatal("unexpected Name")
		}
		if ev[idx].Time.Before(ev[idx-1].Time) {
			t.Fatal("unexpected Time")
		}
	}
	if ev[last].Duration <= 0 {
		t.Fatal("unexpected Duration")
	}
	if ev[last].Err != nil {
		t.Fatal("unexpected Err")
	}
	if ev[last].Name != "tls_handshake_done" {
		t.Fatal("unexpected Name")
	}
	if ev[last].TLSCipherSuite == "" {
		t.Fatal("unexpected TLSCipherSuite")
	}
	if ev[last].TLSNegotiatedProto != "h2" {
		t.Fatal("unexpected TLSNegotiatedProto")
	}
	if !reflect.DeepEqual(ev[last].TLSNextProtos, nextprotos) {
		t.Fatal("unexpected TLSNextProtos")
	}
	if ev[last].TLSPeerCerts == nil {
		t.Fatal("unexpected TLSPeerCerts")
	}
	if ev[last].TLSServerName != "www.google.com" {
		t.Fatal("unexpected TLSServerName")
	}
	if ev[last].TLSVersion == "" {
		t.Fatal("unexpected TLSVersion")
	}
	if ev[last].Time.Before(ev[last-1].Time) {
		t.Fatal("unexpected Time")
	}
}

func TestSaverTLSHandshakerSuccess(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	nextprotos := []string{"h2"}
	saver := &trace.Saver{}
	tlsdlr := &netxlite.TLSDialer{
		Config: &tls.Config{NextProtos: nextprotos},
		Dialer: new(net.Dialer),
		TLSHandshaker: tlsdialer.SaverTLSHandshaker{
			TLSHandshaker: &netxlite.TLSHandshakerStdlib{},
			Saver:         saver,
		},
	}
	conn, err := tlsdlr.DialTLSContext(context.Background(), "tcp", "www.google.com:443")
	if err != nil {
		t.Fatal(err)
	}
	conn.Close()
	ev := saver.Read()
	if len(ev) != 2 {
		t.Fatal("unexpected number of events")
	}
	if ev[0].Name != "tls_handshake_start" {
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
		t.Fatal("unexpected Err")
	}
	if ev[1].Name != "tls_handshake_done" {
		t.Fatal("unexpected Name")
	}
	if ev[1].TLSCipherSuite == "" {
		t.Fatal("unexpected TLSCipherSuite")
	}
	if ev[1].TLSNegotiatedProto != "h2" {
		t.Fatal("unexpected TLSNegotiatedProto")
	}
	if !reflect.DeepEqual(ev[1].TLSNextProtos, nextprotos) {
		t.Fatal("unexpected TLSNextProtos")
	}
	if ev[1].TLSPeerCerts == nil {
		t.Fatal("unexpected TLSPeerCerts")
	}
	if ev[1].TLSServerName != "www.google.com" {
		t.Fatal("unexpected TLSServerName")
	}
	if ev[1].TLSVersion == "" {
		t.Fatal("unexpected TLSVersion")
	}
	if ev[1].Time.Before(ev[0].Time) {
		t.Fatal("unexpected Time")
	}
}

func TestSaverTLSHandshakerHostnameError(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	saver := &trace.Saver{}
	tlsdlr := &netxlite.TLSDialer{
		Dialer: new(net.Dialer),
		TLSHandshaker: tlsdialer.SaverTLSHandshaker{
			TLSHandshaker: &netxlite.TLSHandshakerStdlib{},
			Saver:         saver,
		},
	}
	conn, err := tlsdlr.DialTLSContext(
		context.Background(), "tcp", "wrong.host.badssl.com:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
	for _, ev := range saver.Read() {
		if ev.Name != "tls_handshake_done" {
			continue
		}
		if ev.NoTLSVerify == true {
			t.Fatal("expected NoTLSVerify to be false")
		}
		if len(ev.TLSPeerCerts) < 1 {
			t.Fatal("expected at least a certificate here")
		}
	}
}

func TestSaverTLSHandshakerInvalidCertError(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	saver := &trace.Saver{}
	tlsdlr := &netxlite.TLSDialer{
		Dialer: new(net.Dialer),
		TLSHandshaker: tlsdialer.SaverTLSHandshaker{
			TLSHandshaker: &netxlite.TLSHandshakerStdlib{},
			Saver:         saver,
		},
	}
	conn, err := tlsdlr.DialTLSContext(
		context.Background(), "tcp", "expired.badssl.com:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
	for _, ev := range saver.Read() {
		if ev.Name != "tls_handshake_done" {
			continue
		}
		if ev.NoTLSVerify == true {
			t.Fatal("expected NoTLSVerify to be false")
		}
		if len(ev.TLSPeerCerts) < 1 {
			t.Fatal("expected at least a certificate here")
		}
	}
}

func TestSaverTLSHandshakerAuthorityError(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	saver := &trace.Saver{}
	tlsdlr := &netxlite.TLSDialer{
		Dialer: new(net.Dialer),
		TLSHandshaker: tlsdialer.SaverTLSHandshaker{
			TLSHandshaker: &netxlite.TLSHandshakerStdlib{},
			Saver:         saver,
		},
	}
	conn, err := tlsdlr.DialTLSContext(
		context.Background(), "tcp", "self-signed.badssl.com:443")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if conn != nil {
		t.Fatal("expected nil conn here")
	}
	for _, ev := range saver.Read() {
		if ev.Name != "tls_handshake_done" {
			continue
		}
		if ev.NoTLSVerify == true {
			t.Fatal("expected NoTLSVerify to be false")
		}
		if len(ev.TLSPeerCerts) < 1 {
			t.Fatal("expected at least a certificate here")
		}
	}
}

func TestSaverTLSHandshakerNoTLSVerify(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
	saver := &trace.Saver{}
	tlsdlr := &netxlite.TLSDialer{
		Config: &tls.Config{InsecureSkipVerify: true},
		Dialer: new(net.Dialer),
		TLSHandshaker: tlsdialer.SaverTLSHandshaker{
			TLSHandshaker: &netxlite.TLSHandshakerStdlib{},
			Saver:         saver,
		},
	}
	conn, err := tlsdlr.DialTLSContext(
		context.Background(), "tcp", "self-signed.badssl.com:443")
	if err != nil {
		t.Fatal(err)
	}
	if conn == nil {
		t.Fatal("expected non-nil conn here")
	}
	conn.Close()
	for _, ev := range saver.Read() {
		if ev.Name != "tls_handshake_done" {
			continue
		}
		if ev.NoTLSVerify != true {
			t.Fatal("expected NoTLSVerify to be true")
		}
		if len(ev.TLSPeerCerts) < 1 {
			t.Fatal("expected at least a certificate here")
		}
	}
}

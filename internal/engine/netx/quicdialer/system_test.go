package quicdialer_test

import (
	"context"
	"crypto/tls"
	"testing"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/quicdialer"
	"github.com/ooni/probe-cli/v3/internal/engine/netx/trace"
	"github.com/ooni/probe-cli/v3/internal/errorsx"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func TestSystemDialerSuccessWithReadWrite(t *testing.T) {
	// This is the most common use case for collecting reads, writes
	tlsConf := &tls.Config{
		NextProtos: []string{"h3"},
		ServerName: "www.google.com",
	}
	saver := &trace.Saver{}
	systemdialer := &netxlite.QUICDialerQUICGo{
		QUICListener: &quicdialer.QUICListenerSaver{
			QUICListener: &netxlite.QUICListenerStdlib{},
			Saver:        saver,
		},
	}
	_, err := systemdialer.DialContext(context.Background(), "udp",
		"216.58.212.164:443", tlsConf, &quic.Config{})
	if err != nil {
		t.Fatal(err)
	}
	ev := saver.Read()
	if len(ev) < 2 {
		t.Fatal("unexpected number of events")
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
		case errorsx.ReadFromOperation, errorsx.WriteToOperation:
		default:
			t.Fatal("unexpected Name")
		}
		if ev[idx].Time.Before(ev[idx-1].Time) {
			t.Fatal("unexpected Time", ev[idx].Time, ev[idx-1].Time)
		}
	}
}

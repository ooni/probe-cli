package tracex

import (
	"context"
	"crypto/tls"
	"errors"
	"net"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/model/mocks"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	"github.com/ooni/probe-cli/v3/internal/netxlite/quictesting"
)

type MockDialer struct {
	Dialer model.QUICDialer
	Sess   quic.EarlyConnection
	Err    error
}

func (d MockDialer) DialContext(ctx context.Context, network, host string,
	tlsCfg *tls.Config, cfg *quic.Config) (quic.EarlyConnection, error) {
	if d.Dialer != nil {
		return d.Dialer.DialContext(ctx, network, host, tlsCfg, cfg)
	}
	return d.Sess, d.Err
}

func TestHandshakeSaverSuccess(t *testing.T) {
	nextprotos := []string{"h3"}
	servername := quictesting.Domain
	tlsConf := &tls.Config{
		NextProtos: nextprotos,
		ServerName: servername,
	}
	saver := &Saver{}
	dlr := QUICHandshakeSaver{
		QUICDialer: &netxlite.QUICDialerQUICGo{
			QUICListener: &netxlite.QUICListenerStdlib{},
		},
		Saver: saver,
	}
	sess, err := dlr.DialContext(context.Background(), "udp",
		quictesting.Endpoint("443"), tlsConf, &quic.Config{})
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
	if ev[0].TLSServerName != quictesting.Domain {
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
	if ev[1].TLSServerName != quictesting.Domain {
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
	saver := &Saver{}
	dlr := QUICHandshakeSaver{
		QUICDialer: &netxlite.QUICDialerQUICGo{
			QUICListener: &netxlite.QUICListenerStdlib{},
		},
		Saver: saver,
	}
	sess, err := dlr.DialContext(context.Background(), "udp",
		quictesting.Endpoint("443"), tlsConf, &quic.Config{})
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
		if !strings.HasSuffix(ev.Err.Error(), "tls: handshake failure") {
			t.Fatal("unexpected error", ev.Err)
		}
	}
}

func TestQUICListenerSaverCannotListen(t *testing.T) {
	expected := errors.New("mocked error")
	qls := &QUICListenerSaver{
		QUICListener: &mocks.QUICListener{
			MockListen: func(addr *net.UDPAddr) (model.UDPLikeConn, error) {
				return nil, expected
			},
		},
		Saver: &Saver{},
	}
	pconn, err := qls.Listen(&net.UDPAddr{
		IP:   []byte{},
		Port: 8080,
		Zone: "",
	})
	if !errors.Is(err, expected) {
		t.Fatal("unexpected error", err)
	}
	if pconn != nil {
		t.Fatal("expected nil pconn here")
	}
}

func TestSystemDialerSuccessWithReadWrite(t *testing.T) {
	// This is the most common use case for collecting reads, writes
	tlsConf := &tls.Config{
		NextProtos: []string{"h3"},
		ServerName: quictesting.Domain,
	}
	saver := &Saver{}
	systemdialer := &netxlite.QUICDialerQUICGo{
		QUICListener: &QUICListenerSaver{
			QUICListener: &netxlite.QUICListenerStdlib{},
			Saver:        saver,
		},
	}
	_, err := systemdialer.DialContext(context.Background(), "udp",
		quictesting.Endpoint("443"), tlsConf, &quic.Config{})
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
		case netxlite.ReadFromOperation, netxlite.WriteToOperation:
		default:
			t.Fatal("unexpected Name")
		}
		if ev[idx].Time.Before(ev[idx-1].Time) {
			t.Fatal("unexpected Time", ev[idx].Time, ev[idx-1].Time)
		}
	}
}

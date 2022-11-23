package tlsproxy

import (
	"crypto/tls"
	"errors"
	"net"
	"sync"
	"testing"

	"github.com/ooni/probe-cli/v3/internal/cmd/jafar/uncensored"
)

func TestPass(t *testing.T) {
	listener := newproxy(t, "ooni.io")
	checkdialtls(t, listener.Addr().String(), true, &tls.Config{
		ServerName: "example.com",
	})
	killproxy(t, listener)
}

func TestBlock(t *testing.T) {
	listener := newproxy(t, "ooni.io")
	checkdialtls(t, listener.Addr().String(), false, &tls.Config{
		ServerName: "api.ooni.io",
	})
	killproxy(t, listener)
}

func TestNoSNI(t *testing.T) {
	listener := newproxy(t, "ooni.io")
	checkdialtls(t, listener.Addr().String(), false, &tls.Config{
		ServerName: "",
	})
	killproxy(t, listener)
}

func TestInvalidDomain(t *testing.T) {
	listener := newproxy(t, "ooni.io")
	checkdialtls(t, listener.Addr().String(), false, &tls.Config{
		ServerName: "antani.local",
	})
	killproxy(t, listener)
}

func TestFailHandshake(t *testing.T) {
	listener := newproxy(t, "ooni.io")
	checkdialtls(t, listener.Addr().String(), false, &tls.Config{
		ServerName: "expired.badssl.com",
	})
	killproxy(t, listener)
}

func TestFailConnectingToSelf(t *testing.T) {
	proxy := &CensoringProxy{
		dial: func(network string, address string) (net.Conn, error) {
			return &mockedConnWriteError{}, nil
		},
	}
	listener, err := proxy.Start("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if listener == nil {
		t.Fatal("expected non nil listener here")
	}
	checkdialtls(t, listener.Addr().String(), false, &tls.Config{
		ServerName: "www.google.com",
	})
	killproxy(t, listener)
}

func TestFailWriteAfterConnect(t *testing.T) {
	proxy := &CensoringProxy{
		dial: func(network string, address string) (net.Conn, error) {
			return &mockedConnWriteError{
				// must be different or it refuses connecting to self
				localIP:  net.IPv4(127, 0, 0, 1),
				remoteIP: net.IPv4(127, 0, 0, 2),
			}, nil
		},
	}
	listener, err := proxy.Start("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	if listener == nil {
		t.Fatal("expected non nil listener here")
	}
	checkdialtls(t, listener.Addr().String(), false, &tls.Config{
		ServerName: "www.google.com",
	})
	killproxy(t, listener)
}

func TestListenError(t *testing.T) {
	proxy := NewCensoringProxy(
		[]string{""}, uncensored.NewClient("https://1.1.1.1/dns-query"), "443",
	)
	listener, err := proxy.Start("8.8.8.8:80")
	if err == nil {
		t.Fatal("expected an error here")
	}
	if listener != nil {
		t.Fatal("expected nil listener here")
	}
}

func newproxy(t *testing.T, blocked string) net.Listener {
	proxy := NewCensoringProxy(
		[]string{blocked}, uncensored.NewClient("https://1.1.1.1/dns-query"), "443",
	)
	listener, err := proxy.Start("127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	return listener
}

func killproxy(t *testing.T, listener net.Listener) {
	err := listener.Close()
	if err != nil {
		t.Fatal(err)
	}
}

func checkdialtls(
	t *testing.T, proxyAddr string, expectSuccess bool, config *tls.Config,
) {
	conn, err := tls.Dial("tcp", proxyAddr, config)
	if err != nil && expectSuccess {
		t.Fatal(err)
	}
	if err == nil && !expectSuccess {
		t.Fatal("expected failure here")
	}
	if conn == nil && expectSuccess {
		t.Fatal("expected actionable conn")
	}
	if conn != nil && !expectSuccess {
		t.Fatal("expected nil conn")
	}
	if conn != nil {
		conn.Close()
	}
}

type mockedConnWriteError struct {
	net.Conn
	localIP  net.IP
	remoteIP net.IP
}

func (c *mockedConnWriteError) Write(b []byte) (int, error) {
	return 0, errors.New("cannot write sorry")
}

func (c *mockedConnWriteError) LocalAddr() net.Addr {
	return &net.TCPAddr{
		IP: c.localIP,
	}
}

func (c *mockedConnWriteError) RemoteAddr() net.Addr {
	return &net.TCPAddr{
		IP: c.remoteIP,
	}
}

func TestForwardWriteError(t *testing.T) {
	var wg sync.WaitGroup
	wg.Add(1)
	forward(&wg, &mockedConnReadOkay{}, &mockedConnWriteError{})
}

type mockedConnReadOkay struct {
	net.Conn
}

func (c *mockedConnReadOkay) Read(b []byte) (int, error) {
	return len(b), nil
}

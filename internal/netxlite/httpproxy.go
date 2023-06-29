package netxlite

import (
	"context"
	"errors"
	"fmt"
	"github.com/ooni/probe-cli/v3/internal/model"
	"golang.org/x/net/proxy"
	"net"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// A HttpDialer holds HTTP-specific options
// Specifically for HTTP proxy, we build an HTTP tunnel
type HttpDialer struct {
	proxy.Dialer
	proxyNetwork string
	proxyAddress string
	timeout      time.Duration
	ProxyDial    func(context.Context, string, string) (net.Conn, error)
}

func (d *HttpDialer) Dial(network, address string) (net.Conn, error) {
	if err := d.validateTarget(network, address); err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: err}
	}
	var err error
	var c net.Conn

	if d.ProxyDial != nil {
		c, err = d.ProxyDial(context.Background(), d.proxyNetwork, d.proxyAddress)
	} else {
		nd := &net.Dialer{Timeout: d.timeout}
		c, err = nd.DialContext(context.Background(), d.proxyNetwork, d.proxyAddress)
	}

	if err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: err}
	}
	if err := d.DialWithConn(context.Background(), c, network, address); err != nil {
		c.Close()
		return nil, err
	}
	return c, nil
}

func (d *HttpDialer) validateTarget(network, address string) error {
	switch network {
	case "tcp", "tcp6", "tcp4":
	default:
		return errors.New("network not implemented")
	}
	return nil
}

type Addr struct {
	network string
	Name    string // fully-qualified domain name
	IP      net.IP
	Port    int
}

func (a *Addr) Network() string { return a.network }

func (a *Addr) String() string {
	if a == nil {
		return "<nil>"
	}
	port := strconv.Itoa(a.Port)
	if a.IP == nil {
		return net.JoinHostPort(a.Name, port)
	}
	return net.JoinHostPort(a.IP.String(), port)
}

func splitHostPort(address string) (string, int, error) {
	host, port, err := net.SplitHostPort(address)
	if err != nil {
		return "", 0, err
	}
	portnum, err := strconv.Atoi(port)
	if err != nil {
		return "", 0, err
	}
	if 1 > portnum || portnum > 0xffff {
		return "", 0, errors.New("port number out of range " + port)
	}
	return host, portnum, nil
}

func (d *HttpDialer) pathAddrs(address string) (proxy, dst net.Addr, err error) {
	for i, s := range []string{d.proxyAddress, address} {
		host, port, err := splitHostPort(s)
		if err != nil {
			return nil, nil, err
		}
		a := &Addr{Port: port}
		a.IP = net.ParseIP(host)
		if a.IP == nil {
			a.Name = host
		}
		if i == 0 {
			proxy = a
		} else {
			dst = a
		}
	}
	return
}

func (d *HttpDialer) DialWithConn(ctx context.Context, c net.Conn, network, address string) error {
	if err := d.validateTarget(network, address); err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: err}
	}
	if ctx == nil {
		proxy, dst, _ := d.pathAddrs(address)
		return &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: errors.New("nil context")}
	}

	connectReq := fmt.Sprintf("%v %v HTTP/1.1\r\n"+
		"Host: %v\r\n"+
		"Proxy-Connection: keep-alive\r\n"+
		"User-Agent: %v\r\n\r\n", http.MethodConnect, address, address, model.HTTPHeaderUserAgent)

	b := []byte(connectReq)

	n, err := c.Write(b)
	if err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: err}
	}
	if n != len(b) {
		proxy, dst, _ := d.pathAddrs(address)
		return &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: errors.New("not write enough bytes")}
	}

	c.Read(b)
	if err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: err}
	}

	str := string(b)
	if strings.Split(str, " ")[1] != "200" {
		proxy, dst, _ := d.pathAddrs(address)
		return &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: errors.New("cannot establish connection")}
	}

	return nil
}

func (d *HttpDialer) DialContext(ctx context.Context, network, address string) (net.Conn, error) {
	if err := d.validateTarget(network, address); err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: err}
	}
	if ctx == nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: errors.New("nil context")}
	}

	// proxy dial
	var err error
	var c net.Conn

	if d.ProxyDial != nil {
		c, err = d.ProxyDial(ctx, d.proxyNetwork, d.proxyAddress)
	} else {
		nd := &net.Dialer{Timeout: d.timeout}
		c, err = nd.DialContext(ctx, d.proxyNetwork, d.proxyAddress)
	}
	if err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: err}
	}

	connectReq := fmt.Sprintf("%v %v HTTP/1.1\r\n"+
		"Host: %v\r\n"+
		"Proxy-Connection: keep-alive\r\n"+
		"User-Agent: %v\r\n\r\n", http.MethodConnect, address, address, model.HTTPHeaderUserAgent)

	b := []byte(connectReq)

	n, err := c.Write(b)
	if err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: err}
	}
	if n != len(b) {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: errors.New("not write enough bytes")}
	}

	c.Read(b)
	if err != nil {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: err}
	}

	str := string(b)
	if strings.Split(str, " ")[1] != "200" {
		proxy, dst, _ := d.pathAddrs(address)
		return nil, &net.OpError{Op: http.MethodConnect, Net: network, Source: proxy, Addr: dst, Err: errors.New("cannot establish connection")}
	}

	return c, nil

}

func NewHTTPDialer(network, address string) *HttpDialer {
	return &HttpDialer{
		proxyNetwork: network,
		proxyAddress: address,
		timeout:      dialerDefaultTimeout,
	}
}

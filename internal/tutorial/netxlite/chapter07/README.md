
# Chapter I: HTTP GET with TLS conn

In this chapter we will write together a `main.go` file that
uses netxlite to establish a TLS connection to a remote endpoint
and then fetches a webpage from it using GET.

This file is basically the same as the one used in chapter03
with the small addition of the code to perform the GET.

(This file is auto-generated from the corresponding source file,
so make sure you don't edit it manually.)

## The main.go file

We define `main.go` file using `package main`.

The beginning of the program is equal to chapter03,
so there is not much to say about it.

```Go
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"net"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
	utls "gitlab.com/yawning/utls.git"
)

func main() {
	log.SetLevel(log.DebugLevel)
	address := flag.String("address", "8.8.4.4:443", "Remote endpoint address")
	sni := flag.String("sni", "dns.google", "SNI to use")
	timeout := flag.Duration("timeout", 60*time.Second, "Timeout")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	config := &tls.Config{
		ServerName: *sni,
		NextProtos: []string{"h2", "http/1.1"},
		RootCAs:    nil,
	}
	conn, err := dialTLS(ctx, *address, config)
	if err != nil {
		fatal(err)
	}
	log.Infof("Conn type  : %T", conn)
```

This is where things diverge. We create an HTTP client
using a transport created with `netxlite.NewHTTPTransport`.

This transport will have as TCP connections dialer a
"null" dialer that fails whenever you attempt to dial
(and we should not be dialing anything here since we
already have a TLS connection).

It will also use as TLSDialer (the type that dials TLS
and, morally, combines `dialTCP` with `handshakeTLS`) one
that is "single use". What does this mean? Well, we
create such a TLSDialer using the connection we already
established. The first time the HTTP code dials for
TLS, the TLSDialer will return the connection we passed
to its constructor immediately. Every subsequent TLS
dial attempt will fail.

The result is an HTTPTransport suitable for performing
a single request using the given TLS conn.

(A similar construct allows to create an HTTPTransport that
uses a cleartext TCP connection. In the next chapter we'll
see how to do the same using QUIC.)

TODO(https://github.com/ooni/probe/issues/2534): here we're using the QUIRKY netxlite.NewHTTPTransport
function, but we can probably avoid using it, given that this code is
not using tracing and does not care about those quirks.
```Go
	clnt := &http.Client{Transport: netxlite.NewHTTPTransport(
		log.Log, netxlite.NewNullDialer(),
		netxlite.NewSingleUseTLSDialer(conn),
	)}
```

Once we have the proper transport and client, the rest of
the code is basically standard Go for fetching a webpage
using the GET method.

```Go
	log.Infof("Transport  : %T", clnt.Transport)
	defer clnt.CloseIdleConnections()
	resp, err := clnt.Get(
		(&url.URL{Scheme: "https", Host: *sni, Path: "/"}).String())
	if err != nil {
		fatal(err)
	}
	log.Infof("Status code: %d", resp.StatusCode)
	resp.Body.Close()
}

```

We won't comment on the rest of the program because it is
exactly like what we've seen in chapter03.

```Go

func dialTCP(ctx context.Context, address string) (net.Conn, error) {
	netx := &netxlite.Netx{}
	d := netx.NewDialerWithoutResolver(log.Log)
	return d.DialContext(ctx, "tcp", address)
}

func handshakeTLS(ctx context.Context, tcpConn net.Conn, config *tls.Config) (model.TLSConn, error) {
	netx := &netxlite.Netx{}
	th := netx.NewTLSHandshakerUTLS(log.Log, &utls.HelloFirefox_55)
	return th.Handshake(ctx, tcpConn, config)
}

func dialTLS(ctx context.Context, address string, config *tls.Config) (model.TLSConn, error) {
	tcpConn, err := dialTCP(ctx, address)
	if err != nil {
		return nil, err
	}
	tlsConn, err := handshakeTLS(ctx, tcpConn, config)
	if err != nil {
		tcpConn.Close()
		return nil, err
	}
	return tlsConn, nil
}

func fatal(err error) {
	var ew *netxlite.ErrWrapper
	if !errors.As(err, &ew) {
		log.Fatal("cannot get ErrWrapper")
	}
	log.Warnf("error string    : %s", err.Error())
	log.Warnf("OONI failure    : %s", ew.Failure)
	log.Warnf("failed operation: %s", ew.Operation)
	log.Warnf("underlying error: %+v", ew.WrappedErr)
	os.Exit(1)
}

```

## Running the code

### Vanilla run

You can now run this code as follows:

```bash
go run -race ./internal/tutorial/netxlite/chapter07
```

You will see debug logs describing what is happening along with timing info.

### Connect timeout

```bash
go run -race ./internal/tutorial/netxlite/chapter07 -address 8.8.4.4:1
```

should cause a connect timeout error. Try lowering the timout adding, e.g.,
the `-timeout 5s` flag to the command line.

### Connection refused

```bash
go run -race ./internal/tutorial/netxlite/chapter07 -address '[::1]:1'
```

should give you a connection refused error in most cases. (We are quoting
the `::1` IPv6 address using `[` and `]` here.)

### SNI mismatch

```bash
go run -race ./internal/tutorial/netxlite/chapter07 -sni example.com
```

should give you a TLS invalid hostname error (for historical reasons
named `ssl_invalid_hostname`).

## Conclusions

We have seen how to establish a TLS connection with a website
and then how to GET a webpage using such a connection.

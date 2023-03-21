
# Chapter I: HTTP GET with QUIC conn

In this chapter we will write together a `main.go` file that
uses netxlite to establish a QUIC connection to a remote endpoint
and then fetches a webpage from it using GET.

This file is basically the same as the one used in chapter04
with the small addition of the code to perform the GET.

(This file is auto-generated from the corresponding source file,
so make sure you don't edit it manually.)

## The main.go file

We define `main.go` file using `package main`.

The beginning of the program is equal to chapter04,
so there is not much to say about it.

```Go
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"net/http"
	"net/url"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/lucas-clemente/quic-go"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
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
		NextProtos: []string{"h3"},
		RootCAs:    nil,
	}
	qconn, _, err := dialQUIC(ctx, *address, config)
	if err != nil {
		fatal(err)
	}
	log.Infof("Connection type  : %T", qconn)
```

This is where things diverge. We create an HTTP client
using a transport created with `netxlite.NewHTTP3Transport`.

This transport will use a "single use" QUIC dialer.
What does this mean? Well, we create such a QUICDialer
using the connection we already established. The first
time the HTTP code dials for QUIC, the QUICDialer will
return the connection we passed to its constructor
immediately. Every subsequent QUIC dial attempt will fail.

The result is an HTTPTransport suitable for performing
a single request using the given QUIC conn.

(A similar construct allows to create an HTTPTransport that
uses a cleartext TCP connection. In the previous chapter we've
seen how to do the same using TLS conns.)

```Go
	clnt := &http.Client{Transport: netxlite.NewHTTP3Transport(
		log.Log, netxlite.NewSingleUseQUICDialer(qconn), &tls.Config{},
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
exactly like what we've seen in chapter04.

```Go

func dialQUIC(ctx context.Context, address string,
	config *tls.Config) (quic.EarlyConnection, tls.ConnectionState, error) {
	ql := netxlite.NewQUICListener()
	d := netxlite.NewQUICDialerWithoutResolver(ql, log.Log)
	qconn, err := d.DialContext(ctx, address, config, &quic.Config{})
	if err != nil {
		return nil, tls.ConnectionState{}, err
	}
	return qconn, qconn.ConnectionState().TLS.ConnectionState, nil
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
go run -race ./internal/tutorial/netxlite/chapter08
```

You will see debug logs describing what is happening along with timing info.

### QUIC handshake timeout

```bash
go run -race ./internal/tutorial/netxlite/chapter08 -address 8.8.4.4:1
```

should cause a QUIC handshake timeout error. Try lowering the timout adding, e.g.,
the `-timeout 5s` flag to the command line.

### SNI mismatch

```bash
go run -race ./internal/tutorial/netxlite/chapter08 -sni example.com
```

should give you an error mentioning the certificate is invalid.

## Conclusions

We have seen how to establish a QUIC connection with a website
and then how to GET a webpage using such a connection.

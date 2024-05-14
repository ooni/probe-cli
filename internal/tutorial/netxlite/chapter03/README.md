
# Chapter I: TLS parroting

In this chapter we will write together a `main.go` file that
uses netxlite to establish a new TCP connection and then performs
a TLS handshake using the established connection.

Rather than using the Go standard library, like we did in the
previous chapter, we will use the `gitlab.com/yawning/utls.git`
library to customize the ClientHello to look like Firefox.

(This file is auto-generated from the corresponding source file,
so make sure you don't edit it manually.)

## The main.go file

We define `main.go` file using `package main`.

The beginning of the program is equal to the previous chapter,
so there is not much to say about it.

```Go
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
	"net"
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
	tlsConfig := &tls.Config{ // #nosec G402 - we need to use a large TLS versions range for measuring
		ServerName: *sni,
		NextProtos: []string{"h2", "http/1.1"},
		RootCAs:    nil,
	}
	conn, err := dialTLS(ctx, *address, tlsConfig)
	if err != nil {
		fatal(err)
	}
	state := conn.ConnectionState()
	log.Infof("Conn type          : %T", conn)
	log.Infof("Cipher suite       : %s", netxlite.TLSCipherSuiteString(state.CipherSuite))
	log.Infof("Negotiated protocol: %s", state.NegotiatedProtocol)
	log.Infof("TLS version        : %s", netxlite.TLSVersionString(state.Version))
	_ = conn.Close()
}

func dialTCP(ctx context.Context, address string) (net.Conn, error) {
	netx := &netxlite.Netx{}
	d := netx.NewDialerWithoutResolver(log.Log)
	return d.DialContext(ctx, "tcp", address)
}

func handshakeTLS(ctx context.Context, tcpConn net.Conn, config *tls.Config) (model.TLSConn, error) {
```

The following line of code is where we diverge from the
previous chapter. Here we're creating a TLS handshaker
that uses `gitlab.com/yawning/utls.git` and sets the
ClientHello to look like Firefox 55. (This is also
know as TLS parroting because we're parroting what this
version of Firefox would do.)

Note that, when you use parroting, some settings inside
the `tls.Config` (such as the ALPN) may be ignored
if they conflict with what the parroted browser would do.

```Go
	netx := &netxlite.Netx{}
	th := netx.NewTLSHandshakerUTLS(log.Log, &utls.HelloFirefox_55)
```

The rest of the program is exactly like the one in the
previous chapter, so we won't add further comments.

```Go
	return th.Handshake(ctx, tcpConn, config)
}

func dialTLS(ctx context.Context, address string, config *tls.Config) (model.TLSConn, error) {
	tcpConn, err := dialTCP(ctx, address)
	if err != nil {
		return nil, err
	}
	tlsConn, err := handshakeTLS(ctx, tcpConn, config)
	if err != nil {
		_ = tcpConn.Close()
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

You can now run this code as follows:

```bash
go run -race ./internal/tutorial/netxlite/chapter03
```

You will see debug logs describing what is happening along with timing info.

### Connect timeout

```bash
go run -race ./internal/tutorial/netxlite/chapter03 -address 8.8.4.4:1
```

should cause a connect timeout error. Try lowering the timout adding, e.g.,
the `-timeout 5s` flag to the command line.

### Connection refused

```bash
go run -race ./internal/tutorial/netxlite/chapter03 -address '[::1]:1'
```

should give you a connection refused error in most cases. (We are quoting
the `::1` IPv6 address using `[` and `]` here.)

### SNI mismatch

```bash
go run -race ./internal/tutorial/netxlite/chapter03 -sni example.com
```

should give you a TLS invalid hostname error (for historical reasons
named `ssl_invalid_hostname`).

### TLS handshake reset

If you're on Linux, build Jafar (`go build -v ./internal/cmd/tinyjafar`)
and then run:

```bash
sudo ./tinyjafar -iptables-reset-keyword dns.google
```

Then run in another terminal

```bash
go run ./internal/tutorial/netxlite/chapter03
```

Then you can interrupt Jafar using ^C.

### TLS handshake timeout

If you're on Linux, build Jafar (`go build -v ./internal/cmd/tinyjafar`)
and then run:

```bash
sudo ./tinyjafar -iptables-drop-keyword dns.google
```

Then run in another terminal

```bash
go run ./internal/tutorial/netxlite/chapter03
```

Then you can interrupt Jafar using ^C.

## Conclusions

We have seen how to use netxlite to establish a TCP connection
and perform a TLS handshake using such a connection with a specific
configuration that parrots Firefox v55's ClientHello.

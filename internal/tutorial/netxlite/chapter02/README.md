
# Chapter I: TLS handshakes

In this chapter we will write together a `main.go` file that
uses netxlite to establish a new TCP connection and then performs
a TLS handshake using the established connection.

(This file is auto-generated from the corresponding source file,
so make sure you don't edit it manually.)

## The main.go file

We define `main.go` file using `package main`.

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
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

```

### Main function

```Go
func main() {
```

The beginning of main is just like in the previous chapter
except that here we also have a `-sni` flag.

```Go
	log.SetLevel(log.DebugLevel)
	address := flag.String("address", "8.8.4.4:443", "Remote endpoint address")
	sni := flag.String("sni", "dns.google", "SNI to use")
	timeout := flag.Duration("timeout", 60*time.Second, "Timeout")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
```

We create a TLS config. In general you always want to specify
these three fields when you're performing handshakes:

- `ServerName`, which controls the SNI

- `NextProtos`, which controls the ALPN

- `RootCAs`, which we are forcing here to be the
CA pool bundled with OONI (so we don't have to trust
the system-wide certificate store)

```Go
	tlsConfig := &tls.Config{
		ServerName: *sni,
		NextProtos: []string{"h2", "http/1.1"},
		RootCAs:    netxlite.NewDefaultCertPool(),
	}
```

The logic to dial and handshake have been factored
into a function called `dialTLS`.

```Go
	conn, state, err := dialTLS(ctx, *address, tlsConfig)
```

If there is an error, we bail, like before. Otherwise we
print information about the established TLS connection, which
is returned by `dialTLS` and assigned to `state`. Finally,
like in the previous chapter, we close the connection.

```Go
	if err != nil {
		fatal(err)
	}
	log.Infof("Conn type          : %T", conn)
	log.Infof("Cipher suite       : %s", netxlite.TLSCipherSuiteString(state.CipherSuite))
	log.Infof("Negotiated protocol: %s", state.NegotiatedProtocol)
	log.Infof("TLS version        : %s", netxlite.TLSVersionString(state.Version))
	conn.Close()
}

```

### Dialing and handshaking


The `dialTCP` function is exactly as in the previous chapter.
```Go

func dialTCP(ctx context.Context, address string) (net.Conn, error) {
	d := netxlite.NewDialerWithoutResolver(log.Log)
	return d.DialContext(ctx, "tcp", address)
}

```

The `handshakeTLS` function performs the handshake given a TCP
connection and a TLS config. This function creates a new handshaker
using the stdlib to manage TLS conns (we will see how to use
alternative TLS libraries in the next chapter). Then, once it
has constructed an handshaker, it invokes its `Handshake` method
to obtain a TLS conn (nil on failure), a TLS connection state
(empty on failure), and an error (nil on success).

While the returned connection is a `net.Conn`, the `Handshake`
function guarantees that the returned connection is always
compatible with the `netxlite.TLSConn` interface. Basically
this interface is an extension of `net.Conn` that also
allows to perform TLS specific operations, such as handshaking
and obtaining the connection state. (We will see in a later
chapter why this guarantee helps when writing more complex code.)

```Go

func handshakeTLS(ctx context.Context, tcpConn net.Conn,
	config *tls.Config) (net.Conn, tls.ConnectionState, error) {
	th := netxlite.NewTLSHandshakerStdlib(log.Log)
	return th.Handshake(ctx, tcpConn, config)
}

```

Lastly, `dialTLS` combines `dialTCP` and `handshakeTLS`
together. The code you see here is a stripped down version
of the code in the `measurex` library that helps to
perform this dial+handshake operation in a single function call.

```Go

func dialTLS(ctx context.Context, address string,
	config *tls.Config) (net.Conn, tls.ConnectionState, error) {
	tcpConn, err := dialTCP(ctx, address)
	if err != nil {
		return nil, tls.ConnectionState{}, err
	}
	tlsConn, state, err := handshakeTLS(ctx, tcpConn, config)
	if err != nil {
		tcpConn.Close()
		return nil, tls.ConnectionState{}, err
	}
	return tlsConn, state, nil
}

```

### Printing the error

This code did not change since the previous chapter.

```Go

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
go run -race ./internal/tutorial/netxlite/chapter02
```

You will see debug logs describing what is happening along with timing info.

### Connect timeout

```bash
go run -race ./internal/tutorial/netxlite/chapter02 -address 8.8.4.4:1
```

should cause a connect timeout error. Try lowering the timout adding, e.g.,
the `-timeout 5s` flag to the command line.

### Connection refused

```bash
go run -race ./internal/tutorial/netxlite/chapter02 -address '[::1]:1'
```

should give you a connection refused error in most cases. (We are quoting
the `::1` IPv6 address using `[` and `]` here.)

### SNI mismatch

```bash
go run -race ./internal/tutorial/netxlite/chapter02 -sni example.com
```

should give you a TLS invalid hostname error (for historical reasons
named `ssl_invalid_hostname`).

### TLS handshake reset

If you're on Linux, build Jafar (`go build -v ./internal/cmd/jafar`)
and then run:

```bash
sudo ./jafar -iptables-reset-keyword dns.google
```

Then run in another terminal

```bash
go run ./internal/tutorial/netxlite/chapter02
```

Then you can interrupt Jafar using ^C.

### TLS handshake timeout

If you're on Linux, build Jafar (`go build -v ./internal/cmd/jafar`)
and then run:

```bash
sudo ./jafar -iptables-drop-keyword dns.google
```

Then run in another terminal

```bash
go run ./internal/tutorial/netxlite/chapter02
```

Then you can interrupt Jafar using ^C.

## Conclusions

We have seen how to use netxlite to establish a TCP connection
and perform a TLS handshake using such a connection.

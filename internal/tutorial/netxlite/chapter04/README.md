
# Chapter I: Using QUIC

In this chapter we will write together a `main.go` file that
uses netxlite to establish a new QUIC connection with an UDP endpoint.

Conceptually, this program is very similar to the ones presented
in chapters 2 and 3, except that here we use QUIC.

(This file is auto-generated from the corresponding source file,
so make sure you don't edit it manually.)

## The main.go file

We define `main.go` file using `package main`.

The beginning of the program is equal to the previous chapters,
so there is not much to say about it.

```Go
package main

import (
	"context"
	"crypto/tls"
	"errors"
	"flag"
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
```

The main difference is that we set the ALPN correctly for
QUIC/HTTP3 by using `"h3"` here.

```Go
	config := &tls.Config{
		ServerName: *sni,
		NextProtos: []string{"h3"},
		RootCAs:    nil,
	}
```

Also, where previously we called `dialTLS` now we call
a function with a similar API called `dialQUIC`.

```
	qconn, state, err := dialQUIC(ctx, *address, config)
```

The rest of the main function is pretty much the same.

```Go
	if err != nil {
		fatal(err)
	}
	log.Infof("Connection type          : %T", qconn)
	log.Infof("Cipher suite       : %s", netxlite.TLSCipherSuiteString(state.CipherSuite))
	log.Infof("Negotiated protocol: %s", state.NegotiatedProtocol)
	log.Infof("TLS version        : %s", netxlite.TLSVersionString(state.Version))
	qconn.CloseWithError(0, "")
}

```

The dialQUIC function is new. We need to create a QUIC listener
and, using it, a QUICDialer. These two steps are separated so
higher level code can wrap the QUICDialer and collect stats on
the returned connections. Also, as previously, this dialer is
not attached to a resolver, so it will fail if provided a domain
name. The rationale for doing that is similar to before: we
are focusing on step-by-step measurements where each operation
is performed independently. (That is, we assume that before
the code written in this main we have already resolved the
domain name of interest using a resolver, which we will investigate
in the next two chapters.)

```Go
func dialQUIC(ctx context.Context, address string,
	config *tls.Config) (quic.EarlyConnection, tls.ConnectionState, error) {
	ql := netxlite.NewQUICListener()
	d := netxlite.NewQUICDialerWithoutResolver(ql, log.Log)
	qconn, err := d.DialContext(ctx, address, config, &quic.Config{})
	if err != nil {
		return nil, tls.ConnectionState{}, err
	}
```

The following line unwraps the connection state returned by
QUIC code to be of the same type of the ConnectionState that
we returned in the previous chapters.

```Go
	return qconn, qconn.ConnectionState().TLS.ConnectionState, nil
}

```

The rest of the program is equal to the previous chapters.

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
go run -race ./internal/tutorial/netxlite/chapter04
```

You will see debug logs describing what is happening along with timing info.

### QUIC handshake timeout

```bash
go run -race ./internal/tutorial/netxlite/chapter04 -address 8.8.4.4:1
```

should cause a QUIC timeout error. Try lowering the timout adding, e.g.,
the `-timeout 5s` flag to the command line.

### SNI mismatch

```bash
go run -race ./internal/tutorial/netxlite/chapter04 -sni example.com
```

should give you a TLS error mentioning that the certificate is invalid.

## Conclusions

We have seen how to use netxlite to establish a QUIC connection
with a remote UDP endpoint speaking QUIC.


# Chapter I: Using the "system" DNS resolver

In this chapter we will write together a `main.go` file that
uses the "system" DNS resolver to lookup domain names.

The "system" resolver is the one used by `getaddrinfo` on Unix.

(This file is auto-generated from the corresponding source file,
so make sure you don't edit it manually.)

## The main.go file

We define `main.go` file using `package main`.

```Go
package main

import (
	"context"
	"errors"
	"flag"
	"os"
	"time"

	"github.com/apex/log"
	"github.com/ooni/probe-cli/v3/internal/netxlite"
)

func main() {
```

The beginning of the program is equal to the previous chapters,
so there is not much to say about it.

```Go
	log.SetLevel(log.DebugLevel)
	hostname := flag.String("hostname", "dns.google", "Hostname to resolve")
	timeout := flag.Duration("timeout", 60*time.Second, "Timeout")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
```

We create a new resolver using the standard library to perform
domain name resolutions. Unless you're cross compiling, this
resolver will call the system resolver using a C API. On Unix
the called C API is `getaddrinfo`.

The returned resolver implements an interface that is very
close to the API of the `net.Resolver` struct.

```Go
	reso := netxlite.NewStdlibResolver(log.Log)
```

We call `LookupHost` to map the hostname to IP addrs. The returned
value is either a list of addrs or an error.

```Go
	addrs, err := reso.LookupHost(ctx, *hostname)
	if err != nil {
		fatal(err)
	}
	log.Infof("resolver addrs: %+v", addrs)
}

```

This function is exactly like it was in previous chapters.

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
go run -race ./internal/tutorial/netxlite/chapter05
```

You will see debug logs describing what is happening along with timing info.

### NXDOMAIN error

```bash
go run -race ./internal/tutorial/netxlite/chapter05 -hostname antani.ooni.io
```

should cause a `dns_nxdomain_error`, because the domain does not exist.

### Timeout

```bash
go run -race ./internal/tutorial/netxlite/chapter05 -timeout 10us
```

should cause a timeout error, because the timeout is ridicolously small.

## Conclusions

We have seen how to use the "system" DNS resolver.


# Chapter I: Using a custom UDP resolver

In this chapter we will write together a `main.go` file that
uses a custom UDP DNS resolver to lookup domain names.

This program is very similar to the one in the previous chapter
except that we'll be configuring a custom resolver.

(This file is auto-generated from the corresponding source file,
so make sure you don't edit it manually.)

## The main.go file

We define `main.go` file using `package main`.

There's not much to say about the beginning of the program
since it is equal to the one in the previous chapter.

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
	log.SetLevel(log.DebugLevel)
	hostname := flag.String("hostname", "dns.google", "Hostname to resolve")
	timeout := flag.Duration("timeout", 60*time.Second, "Timeout")
	serverAddr := flag.String("server-addr", "1.1.1.1:53", "DNS server address")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
```

Here's where we start to diverge. We create a dialer without a resolver,
which is going to be used by the UDP resolver.

```Go
	dialer := netxlite.NewDialerWithoutResolver(log.Log)
```

Then, we create an UDP resolver. The arguments are the same as for
creating a system resolver, except that we also need to specify the
UDP endpoint address at which the server is listening.

```Go
	reso := netxlite.NewParallelUDPResolver(log.Log, dialer, *serverAddr)
```

The API we invoke is the same as in the previous chapter, though,
and the rest of the program is equal to the one in the previous chapter.

```Go
	addrs, err := reso.LookupHost(ctx, *hostname)
	if err != nil {
		fatal(err)
	}
	log.Infof("resolver addrs: %+v", addrs)
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
go run -race ./internal/tutorial/netxlite/chapter06
```

You will see debug logs describing what is happening along with timing info.

### NXDOMAIN

```bash
go run -race ./internal/tutorial/netxlite/chapter06 -hostname antani.ooni.io
```

should cause a `dns_nxdomain_error`, because the domain does not exist.

### Timeout

```bash
go run -race ./internal/tutorial/netxlite/chapter06 -timeout 10us
```

should cause a timeout error, because the timeout is ridicolously small.

```bash
go run -race ./internal/tutorial/netxlite/chapter06 -server-addr 1.1.1.1:1
```

should also cause a timeout, because 1.1.1.1:1 is not an endpoint
where a DNS-over-UDP resolver is listening.

## Conclusions

We have seen how to use a custom DNS-over-UDP resolver.

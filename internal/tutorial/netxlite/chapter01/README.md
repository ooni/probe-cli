
# Chapter I: establishing TCP connections

In this chapter we will write together a `main.go` file that
uses netxlite to establish a new TCP connection.

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

We use apex/log and configure it to emit debug messages. This
setting will allow us to see netxlite emitted logs.

```Go
	log.SetLevel(log.DebugLevel)
```

We use the flags package to define command line options and we
parse the command line options with `flag.Parse`.

```Go
	address := flag.String("address", "8.8.4.4:443", "Remote endpoint address")
	timeout := flag.Duration("timeout", 60*time.Second, "Timeout")
	flag.Parse()
```

We use the standard Go idiom to set a timeout using a context.

```Go
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
```

The bulk of the logic has been factored into a `dialTCP` function.

```Go
	conn, err := dialTCP(ctx, *address)
```

If there is a failure we invoke a function that prints the
error that occurred and then calls `os.Exit(1)`

```Go
	if err != nil {
		fatal(err)
	}
```

Otherwise, we're tidy and close the opened connection.

```Go
	conn.Close()
}

```

### Dialing for TCP

We construct a netxlite.Dialer (i.e., a type similar to net.Dialer)
and we use it to dial the new connection.

Note that the dialer we're constructing here is not attached to
a resolver. This means that, if `address` contains a domain name
rather than an IP address, the dial operation will fail.

While it is possible in netxlite to construct a dialer using a
resolver, here we're focusing on the step-by-step measuring perspective
where we want to perform each operation independently.

```Go
func dialTCP(ctx context.Context, address string) (net.Conn, error) {
	d := netxlite.NewDialerWithoutResolver(log.Log)
	return d.DialContext(ctx, "tcp", address)
}

```

### Printing the error

Fundamental netxlite types guarantee that they always return a
`*netxlite.ErrWrapper` type on error. This type is an `error` and
we can use `errors.As` to see its content:

- the Failure field is the OONI error string as specified in
https://github.com/ooni/spec, and is also the string that
is emitted in case one calls `err.Error`;

- Operation is the name of the operation that failed;

- WrappedErr is the underlying error that occurred and has
been wrapped by netxlite.

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
go run -race ./internal/tutorial/netxlite/chapter01
```

You will see debug logs describing what is happening along with timing info.

### Connection timeout

```bash
go run -race ./internal/tutorial/netxlite/chapter01 -address 8.8.4.4:1
```

should cause a connect timeout error. Try lowering the timout adding, e.g.,
the `-timeout 5s` flag to the command line.

### Connection refused

```bash
go run -race ./internal/tutorial/netxlite/chapter01 -address '[::1]:1'
```

should give you a connection refused error in most cases. (We are quoting
the `::1` IPv6 address using `[` and `]` here.)

## Conclusions

We have seen how to use netxlite to establish a TCP connection.

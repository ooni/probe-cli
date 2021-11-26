
# Chapter XI: Measuring a URL

This program shows how to measure an HTTP/HTTPS URL. We
are going to call an API whose implementation is
basically the same code we have seen in the previous
chapter, to obtain an URL measurement in a more compact
way. (As an historical note, the API we are going to
call has indeed been written as a refactoring of
the code we introduced in the previous chapter.)

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measurex/chapter11/main.go`.)

## main.go

The beginning of the program is much simpler. We have removed
our custom measurement type. We are now going to use the
`URLMeasurement` type (`go doc ./internal/measurex.URLMeasurement`),
which has the same fields of `measurement` in chapter10 _plus_
some extra fields that we'll examine in a later chapter.

```Go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func print(v interface{}) {
	data, err := json.Marshal(v)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

func main() {
	URL := flag.String("url", "https://www.google.com/", "URL to fetch")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
```

We create a measurer, cookies, and headers like we
saw in the previous chapter.

```Go
	mx := measurex.NewMeasurerWithDefaultSettings()
	cookies := measurex.NewCookieJar()
	headers := measurex.NewHTTPRequestHeaderForMeasuring()
```

Then we call `MeasureURL`. This function's implementation
is in `./internal/measurex/measurer.go` and is pretty
much a refactoring of the code in chapter10.

The arguments are:

- the context as usual

- the number of parallel goroutines to use to perform parallelizable
operations (passing zero or negative will cause the code to use
a reasonably small default value)

- the unparsed URL to measure

- the headers we want to use

- a jar for cookies

```Go
	const parallelism = 3
	m, err := mx.MeasureURL(ctx, parallelism, *URL, headers, cookies)
```
The return value is either an `URLMeasurement`
or an error. The error happens, for example, if
the input URL scheme is not "http" or "https" (which
we handled by panicking in chapter07).

Now, rather than panicking inside `MeasureURL`, we
return the error to the caller and we `panic`
here on `main` using the `PanicOnError` function.

```Go
	runtimex.PanicOnError(err, "mx.MeasureURL failed")
	print(m)
}

```

## Running the example program

Let us perform a vanilla run first:

```bash
go run -race ./internal/tutorial/measurex/chapter11 | jq
```

Take a look at the JSON output and compare it with:

```bash
go run -race ./internal/tutorial/measurex/chapter10 -url https://www.google.com | jq
```

(which is basically forcing chapter10 to run with the
the default URL we use in this chapter).

Can you explain why we are able to measure more endpoints
in this chapter by checking the implementation of `MeasureURL`
and compare it to the code written in chapter10?

Now run:

```bash
go run -race ./internal/tutorial/measurex/chapter11 -url https://google.com | jq
```

Do you see the opportunity there for following redirections? :^).

## Conclusion

We have introduced `MeasureURL`, the top-level API for
measuring a single URL.


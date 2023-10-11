
# Chapter XII: Following redirections.

This program shows how to combine the URL measurement
"step" introduced in the previous chapter with
following redirections. If we say that the previous
chapter performed a "web step", then we can say
that here we're performing multiple "web steps".

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measurex/chapter12/main.go`.)

## main.go

The beginning of the program is pretty much the
same, except that here we need to define a
`measurement` container type that will contain
the result of each "web step".

```Go
package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"time"

	"github.com/ooni/probe-cli/v3/internal/legacy/measurex"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

type measurement struct {
	URLs []*measurex.ArchivalURLMeasurement
}

func print(v interface{}) {
	data, err := json.Marshal(v)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

func main() {
	URL := flag.String("url", "http://facebook.com/", "URL to fetch")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	all := &measurement{}
	mx := measurex.NewMeasurerWithDefaultSettings()
	cookies := measurex.NewCookieJar()
	headers := measurex.NewHTTPRequestHeaderForMeasuring()
```

Everything above this line is like in chapter11. What changes
now is that we're calling `MeasureURLAndFollowRedirections`
instead of `MeasureURL`.

Rather than returning a single measurement, this function
returns a channel where it posts the result of measuring
the original URL along with all its redirections. Internally,
`MeasureURLAndFollowRedirections` calls `MeasureURL`.

The parallelism argument dictates how many parallel goroutine
to use for parallelizable operations. (A zero or negative
value implies that the code should use a sensible default value.)

We accumulate the results in `URLs` and print `m`. The channel
is closed when done by `MeasureURLAndFollowRedirections`, so we leave the loop.

```Go
	const parallelism = 3
	for m := range mx.MeasureURLAndFollowRedirections(ctx, parallelism, *URL, headers, cookies) {
		all.URLs = append(all.URLs, measurex.NewArchivalURLMeasurement(m))
	}
	print(all)
}

```

## Running the example program

Let us perform a vanilla run first:

```bash
go run -race ./internal/tutorial/measurex/chapter12 | jq
```

Take a look at the JSON. You should see several redirects
and that we measure each endpoint of each redirect, including
QUIC endpoints that we discover on the way.

Exercise: remove code for converting to OONI data format
and compare output with previous chapter. See any difference?

## Conclusion

We have introduced `MeasureURLAndFollowRedirect`, the
top-level API for fully measuring a URL and all the URLs
that derive from such an URL via redirection.


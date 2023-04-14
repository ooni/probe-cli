
# Chapter XIV: A possible rewrite of Web Connectivity

In this chapter we try to solve the exercise laid out in
the previous chapter, using `measurex` primitives.

(This file is auto-generated. Do not edit it directly! To apply
changes you need to modify `./internal/tutorial/measurex/chapter14/main.go`.)

## main.go

The beginning of the file is always pretty much the same.

```Go
package main

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"flag"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/ooni/probe-cli/v3/internal/measurex"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
)

func print(v interface{}) {
	data, err := json.Marshal(v)
	runtimex.PanicOnError(err, "json.Marshal failed")
	fmt.Printf("%s\n", string(data))
}

```

## measurement type

We define a measurement type with the fields
that a Web Connectivity measurement should have.

```Go

type measurement struct {
	Queries       []*measurex.ArchivalDNSLookupEvent        `json:"queries"`
	TCPConnect    []*measurex.ArchivalTCPConnect            `json:"tcp_connect"`
	TLSHandshakes []*measurex.ArchivalQUICTLSHandshakeEvent `json:"tls_handshakes"`
	Requests      []*measurex.ArchivalHTTPRoundTripEvent    `json:"requests"`
}

```

## WebConnectivity implementation

We define a function that takes in input a context and a URL to
measure and returns a measurement or an error.

We will only error out in case the input does not allow us to
proceed (i.e., invalid input URL).

```Go

func webConnectivity(ctx context.Context, URL string) (*measurement, error) {
```

We start by parsing the input URL. If we cannot parse it, of
course this is a hard error and we cannot continue.

```Go
	parsedURL, err := url.Parse(URL)
	if err != nil {
		return nil, err
	}

```

We create an empty measurement and a measurer with
default settings like we did in the previous chapters.

```Go
	m := &measurement{}
	mx := measurex.NewMeasurerWithDefaultSettings()

```

Now it's time to start measuring. We will address all
the points laid out in the previous chapter.

### 1. Enumerating IP addrs

Let us enumerate all the IP addresses for
the input URL's domain using the system resolver.

```Go
	dns := mx.LookupHostSystem(ctx, parsedURL.Hostname())
	m.Queries = append(
		m.Queries, measurex.NewArchivalDNSLookupEventList(dns.LookupHost)...)

```

This is code we have already seen in the previous chapters.


### 2. Building a list of endpoints

```Go
	epnts, err := measurex.AllHTTPEndpointsForURL(parsedURL, http.Header{}, dns)
	if err != nil {
		return nil, err
	}

```

This is also code we have seen in previous chapters. The only
difference is that we supply empty headers since we're not going
to actually use the headers inside the endpoints.

### 3 and 4. Measure each endpoint

We will loop through the endpoints in the previous point
and issue the correct TCP or TLS primitive depending on
whether the input URL is HTTP or HTTPS.

```Go
	for _, epnt := range epnts {
		switch parsedURL.Scheme {
		case "http":
			tcp := mx.TCPConnect(ctx, epnt.Address)
			m.TCPConnect = append(
				m.TCPConnect, measurex.NewArchivalTCPConnectList(tcp.Connect)...)
		case "https":
			config := &tls.Config{
				ServerName: parsedURL.Hostname(),
				NextProtos: []string{"h2", "http/1.1"},
				RootCAs:    nil, // use netxlite's default
			}
			tls := mx.TLSConnectAndHandshake(ctx, epnt.Address, config)
			m.TCPConnect = append(
				m.TCPConnect, measurex.NewArchivalTCPConnectList(tls.Connect)...)
			m.TLSHandshakes = append(m.TLSHandshakes,
				measurex.NewArchivalQUICTLSHandshakeEventList(tls.TLSHandshake)...)
		}
	}

```

At this point we've addressed points 1-4. So let's
now focus on the last point:

### 5. HTTP measurement

We need to manually build a `MeasurementDB`. This is a
"database" where the networking code will store events.

```Go

	db := &measurex.MeasurementDB{}

```

Following the hint from the previous chapter we use the
`NewTracingHTTPTransportWithDefaultSettings` factory
to create an `http.Transport`-like object that will trace
HTTP round trip events writing them into `db`.


```Go

	txp := measurex.NewTracingHTTPTransportWithDefaultSettings(mx.Begin, mx.Logger, db)

```

We now build an `http.Client` using the transport
we've just created and a cookie jar (which we
use because otherwise some redirects will lead
to a redirect loop, as mentioned in previous chapters).

```Go

	clnt := &http.Client{
		Transport: txp,
		Jar:       measurex.NewCookieJar(),
	}

```

Now we use a method of the measurer that allows us to
perform an HTTP GET with an existing HTTP client
and a URL. This method will set a timeout and perform
the round trip. Reading a snapshot of the response
body is not implemented by this function but rather
is a property of the "tracing" HTTP transport we
created above (this type of transport is the one we
have been using internally in all the examples
presented so far.)

```Go

	resp, _ := mx.HTTPClientGET(ctx, clnt, parsedURL)

```

To be tidy, we also close the response body in case
we have a response. We don't really need to read
the body here. As mentioned previously, we're already
using an HTTP transport reading a body snapshot.

```Go

	if resp != nil {
		resp.Body.Close() // tidy
	}

```

Finally, we append the round trips we performed into
the right field and return the measurement.

To this end, we're using the `db.AsMeasurement` method that
takes the current set of events into `db` and assembles
them into the `Measurement` struct we've been using in all
the chapters we have seen so far.

```Go

	m.Requests = append(m.Requests, measurex.NewArchivalHTTPRoundTripEventList(
		db.AsMeasurement().HTTPRoundTrip)...)
	return m, nil
}

```

The rest of the program is pretty straightforward.

```Go

func main() {
	URL := flag.String("url", "https://www.google.com/", "URL to fetch")
	timeout := flag.Duration("timeout", 60*time.Second, "timeout to use")
	flag.Parse()
	ctx, cancel := context.WithTimeout(context.Background(), *timeout)
	defer cancel()
	m, err := webConnectivity(ctx, *URL)
	runtimex.PanicOnError(err, "invalid arguments to webConnectivity (wrong URL?)")
	print(m)
}

```

## Running the example program

Let us perform a vanilla run first:

```bash
go run -race ./internal/tutorial/measurex/chapter14 | jq
```

Take a look at the JSON.

Now try running the program with `http://gmail.com` as
input. Take note of the redirect chain. See how the
domain changes during the redirect. Take note of the
fact that we are not measuring any TLS handshake. See
how we're not trying QUIC endpoints. These are, in
fact, some of the limitations of Web Connectivity that
we were trying to address when we wrote `measurex`.

Also, build the miniooni research client:

```
go build -v ./internal/cmd/miniooni
```

Run Web Connectivity with:

```
./miniooni -ni http://gmail.com web_connectivity
```

This writes the report in a file named `report.jsonl`.

Check the content of the file and match it with the
output of this chapter. Are there other notable
differences between the two outputs?

### Bonus question

The solution we presented is true to the original
spirit of Web Connectivity, where we first perform
separate DNS, TCP/TLS steps, and then we also
perform a separate HTTP step. Is there in `measurex`
an API allowing you to invert the order of the
operations, that is:

1. build a full-fledged HTTP client where we can
trace _any_ operation;

2. use such client to measure the URL;

3. figure out what TCP endpoints we did not
test for TCP/TLS during this process and run
TCP/TLS testing only for them?

If such an API exist, can you write a simple
main.go client that implements points 1-3 above?

## Conclusion

We have presented the solution to the exercise
proposed in the previous chapter, i.e., how
to rewrite Web Connectivity using `measurex` API.

You have now been exposed to some complexity and
APIs to perform OONI measurements. So you should now
be read to help us write new and maitain existing
network experiments.

If you have further questions, please [contact us](
https://ooni.org/about/).


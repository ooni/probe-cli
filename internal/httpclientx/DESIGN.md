# ./internal/httpclientx

This package aims to replace previously existing packages for interact with
backend services controlled either by us or by third parties.

As of 2024-04-22, these packages are:

* `./internal/httpx`, which is currently deprecated;

* `./internal/httpapi`, which aims to automatically generate swaggers from input
and output messages and implements support for falling back.

The rest of this document explains the requirements and describes the design.

## Table of Contents

- [Requirements](#requirements)
- [Design](#design)
	- [Overlapped Operations](#overlapped-operations)
	- [Extensibility](#extensibility)
	- [Functionality Comparison](#functionality-comparison)
		- [GetJSON](#getjson)
		- [GetRaw](#getraw)
		- [GetXML](#getxml)
		- [PostJSON](#postjson)
- [Nil Safety](#nil-safety)
- [Refactoring Plan](#refactoring-plan)
- [Limitations and Future Work](#limitations-and-future-work)

## Requirements

We want this new package to:

1. Implement common access patterns (GET with JSON body, GET with XML body, GET with
raw body, and POST with JSON request and response bodies).

2. Optionally log request and response bodies.

3. Factor common code for accessing such services.

4. Take advantage of Go generics to automatically marshal and unmarshal without
having to write specific functions for each request/response pair.

5. Support some kind of fallback policy like `httpapi` because we used this
for test helpers and, while, as of today, we're mapping serveral TH domain names
to a single IP address, it might still be useful to fallback.

6. Provide an easy way for the caller to know whether an HTTP request failed
and, if so, with which status code, which is needed to intercept `401` responses
to take the appropriate logging-in actions.

7. Make the design extensible such that re-adding unused functionality
does not require us to refactor the code much.

8. Functional equivalent with existing packages (modulo the existing
functionality that is not relevant anymore).

Non goals:

1. automatically generate swaggers from the Go representation of API invocation
for these reasons: (1) the OONI backend swagger is only for documentational
purposes and it is not always in sync with reality; (2) by doing that, we obtained
overly complex code, which hampers maintenance.

2. implementing algorithms such as logging in to the OONI backend and requesting
tokens, which should be the responsibility of another package.

## Design

This package supports the following operations:

```Go
type Config struct {
	Client model.HTTPClient
	Logger model.Logger
	UserAgent string
}

type Endpoint struct {
	URL string
	Host string // optional for cloudfronting
}

func GetJSON[Output any](ctx context.Context, epnt *Endpoint, config *Config) (Output, error)

func GetRaw(ctx context.Context, epnt *Endpoint, config *Config) ([]byte, error)

func GetXML[Output any](ctx context.Context, epnt *Endpoint, config *Config) (Output, error)

func PostJSON[Input, Output any](ctx context.Context, epnt *Endpoint, input Input, config *Config) (Output, error)
```

(The `*Config` is the last argument because it is handy to create it inline when calling
and having it last reduces readability the least.)

These operations implement all the actions listed in the first requirement.

The `Config` struct allows to add new optional fields to implement new functionality without
changing the API and with minimal code refactoring efforts (seventh requirement).

We're using generics to automate marshaling and umarshaling (requirement four).

The internal implementation is such that we reuse code to avoid code duplication,
thus addressing also requirement three.

Additionally, whenever a call fails with a non-200 status code, the return
value can be converted to the following type using `errors.As`:

```Go
type ErrRequestFailed struct {
        StatusCode int
}

func (err *ErrRequestFailed) Error() string
```

Therefore, we have a way to know why a request failed (requirement six).

To avoid logging bodies, one just needs to pass `model.DiscardLogger` as the
`logger` (thus fulfilling requirement two).

### Overlapped Operations

The code at `./internal/httpapi` performs sequential function calls. This design
does not interact well with the `enginenetx` package and its dial tactics. A better
strategy is to allow calls to be overlapped. This means that, if the `enginenetx`
is busy trying tactics for a given API endpoint, we eventually try to use the
subsequent (semantically-equivalent) endpoint after a given time, without waiting
for the first endpoint to complete.

We allow for overlapped operations by defining these constructors:

```Go
func NewOverlappedGetJSON[Output any](config *Config) *Overlapped[Output]

func NewOverlappedGetRaw(config *Config) *Overlapped[[]byte]

func NewOverlappedGetXML[Output any](config *Config) *Overlapped[Output]

func NewOverlappedPostJSON[Input, Output any](input Input, config *Config) *Overlapped[Output]
```

They all construct the same `*Overlapped` struct, which looks like this:

```Go
type Overlapped[Output any] struct {
	RunFunc func(ctx context.Context, epnt *Endpoint) (Output, error)

	ScheduleInterval time.Duration
}
```

The constructor configures `RunFunc` to invoke the call corresponding to the construct
name (i.e., `NewOverlappedGetXML` configures `RunFunc` to run `GetXML`).

Then, we define the following method:

```Go
func (ovx *Overlapped[Output]) Run(ctx context.Context, epnts ...*Endpoint) (Output, error)
```

This method starts N goroutines to issue the API calls with each endpoint URL. (A classic example
is for the URLs to be `https://0.th.ooni.org/`, `https://1.th.ooni.org/` and so on.)

By default, `ScheduleInterval` is 15 seconds. If the first endpoint URL does not provide a result
within 15 seconds, we try the second one. That is, every 15 seconds, we will attempt using
another endpoint URL, until there's a successful response or we run out of URLs.

As soon as we have a successful response, we cancel all the other pending operations
that may exist. Once all operations have terminated, we return to the caller.

### Extensibility

We use the `Config` object to package common settings. Thus adding a new field, only means
the following:

1. Adding a new OPTIONAL field to `Config`.

2. Honoring this field inside the internal implementation.

_Et voilà_, this should allow for minimal efforts API upgrades.

In fact, we used this strategy to easily add support for cloudfront in
[probe-cli#1577](https://github.com/ooni/probe-cli/pull/1577).

### Functionality Comparison

This section compares side-by-side the operations performed by each implementation
as of [probe-cli@7dab5a29812](https://github.com/ooni/probe-cli/tree/7dab5a29812) to
show that they implement ~equivalent functionality. This should be the case, since
the `httpxclientx` package is a refactoring of `httpapi`, which in turn contains code
originally derived from `httpx`. Anyways, better to double check.

#### GetJSON

We compare to `httpapi.Call` and `httpx.GetJSONWithQuery`.

| Operation                 | GetJSON | httpapi | httpx |
| ------------------------- | ------- | ------- | ----- |
| enforce a call timeout    |   NO    |   yes   |  NO   |
| parse base URL            |   NO    |   yes   |  yes  |
| join path and base URL    |   NO    |   yes   |  yes  |
| append query to URL       |   NO    |   yes   |  yes  |
| NewRequestWithContext     |   yes️   |   yes   |  yes  |
| handle cloud front        |   yes   |   yes   |  yes  |
| set Authorization         |   yes   |   yes   |  yes  |
| set Accept                |   NO    |   yes   |  yes  |
| set User-Agent            |   yes ️  |   yes   |  yes  |
| set Accept-Encoding gzip  |   yes️   |   yes   |  NO   |
| (HTTPClient).Do()         |   yes   |   yes   |  yes  |
| defer resp.Body.Close()   |   yes   |   yes   |  yes  |
| handle gzip encoding      |   yes   |   yes   |  NO   |
| limit io.Reader           |   yes   |   yes   |  yes  |
| netxlite.ReadAllContext() |   yes   |   yes   |  yes  |
| handle truncated body     |   yes   |   yes   |  NO   |
| log response body         |   yes   |   yes   |  yes  |
| handle non-200 response   | ️  yes   |   yes*  |  yes* |
| unmarshal JSON            |   yes   |   yes   |  yes  |

The `yes*` means that `httpapi` rejects responses with status codes `>= 400` (like cURL)
while the new package only accepts status codes `== 200`. This difference should be of little
practical significance for all the APIs we invoke and the new behavior is stricter.

Regarding all the other cases for which `GetJSON` is marked as "NO":

1. Enforcing a call timeout is better done just through the context like `httpx` does.

2. `GetJSON` lets the caller completely manage the construction of the URL, so we do not need
code to join together a base URL, possibly including a base path, a path, and a query (and we're
introducing the new `./internal/urlx` package to handle this situation).

3. Setting the `Accept` header does not seem to matter in out context because we mostly
call API for which there's no need for content negotiation.

#### GetRaw

Here we're comparing to `httpapi.Call` and `httpx.FetchResource`.

| Operation                 | GetRaw  | httpapi | httpx |
| ------------------------- | ------- | ------- | ----- |
| enforce a call timeout    |   NO    |   yes   |  NO   |
| parse base URL            |   NO    |   yes   |  yes  |
| join path and base URL    |   NO    |   yes   |  yes  |
| append query to URL       |   NO    |   yes   |  yes  |
| NewRequestWithContext     |   yes️   |   yes   |  yes  |
| handle cloud front        |   yes   |   yes   |  yes  |
| set Authorization         |   yes   |   yes   |  yes  |
| set Accept                |   NO    |   yes   |  yes  |
| set User-Agent            |   yes ️  |   yes   |  yes  |
| set Accept-Encoding gzip  |   yes️   |   yes   |  NO   |
| (HTTPClient).Do()         |   yes   |   yes   |  yes  |
| defer resp.Body.Close()   |   yes   |   yes   |  yes  |
| handle gzip encoding      |   yes   |   yes   |  NO   |
| limit io.Reader           |   yes   |   yes   |  yes  |
| netxlite.ReadAllContext() |   yes   |   yes   |  yes  |
| handle truncated body     |   yes   |   yes   |  NO   |
| log response body         |   yes   |   yes   |  yes  |
| handle non-200 response   | ️  yes   |   yes*  |  yes  |

Here we can basically make equivalent remarks as those of the previous section.

#### GetXML

There's no direct equivalent of `GetXML` in `httpapi` and `httpx`. Therefore, when using these
two APIs, the caller would need to fetch a raw body and then manually parse XML.

| Operation                 | GetXML  | httpapi | httpx |
| ------------------------- | ------- | ------- | ----- |
| enforce a call timeout    |   NO    |   N/A   |  N/A  |
| parse base URL            |   NO    |   N/A   |  N/A  |
| join path and base URL    |   NO    |   N/A   |  N/A  |
| append query to URL       |   NO    |   N/A   |  N/A  |
| NewRequestWithContext     |   yes️   |   N/A   |  N/A  |
| handle cloud front        |   yes   |   N/A   |  N/A  |
| set Authorization         |   yes   |   N/A   |  N/A  |
| set Accept                |   NO    |   N/A   |  N/A  |
| set User-Agent            |   yes ️  |   N/A   |  N/A  |
| set Accept-Encoding gzip  |   yes️   |   N/A   |  N/A  |
| (HTTPClient).Do()         |   yes   |   N/A   |  N/A  |
| defer resp.Body.Close()   |   yes   |   N/A   |  N/A  |
| handle gzip encoding      |   yes   |   N/A   |  N/A  |
| limit io.Reader           |   yes   |   N/A   |  N/A  |
| netxlite.ReadAllContext() |   yes   |   N/A   |  N/A  |
| handle truncated body     |   yes   |   N/A   |  N/A  |
| log response body         |   yes   |   N/A   |  N/A  |
| handle non-200 response   | ️  yes   |   N/A   |  N/A  |
| unmarshal XML             |   yes   |   N/A   |  N/A  |

Because comparison is not possible, there is not much else to say.

#### PostJSON

Here we're comparing to `httpapi.Call` and `httpx.PostJSON`.

| Operation                 | PostJSON | httpapi | httpx |
| ------------------------- | -------- | ------- | ----- |
| marshal JSON              |   yes    |   yes~  |  yes  |
| log request body          |   yes    |   yes   |  yes  |
| enforce a call timeout    |   NO     |   yes   |  NO   |
| parse base URL            |   NO     |   yes   |  yes  |
| join path and base URL    |   NO     |   yes   |  yes  |
| append query to URL       |   NO     |   yes   |  yes  |
| NewRequestWithContext     |   yes️    |   yes   |  yes  |
| handle cloud front        |   yes    |   yes   |  yes  |
| set Authorization         |   yes    |   yes   |  yes  |
| set Accept                |   NO     |   yes   |  yes  |
| set Content-Type          |   yes    |   yes   |  yes  |
| set User-Agent            |   yes ️   |   yes   |  yes  |
| set Accept-Encoding gzip  |   yes️    |   yes   |  NO   |
| (HTTPClient).Do()         |   yes    |   yes   |  yes  |
| defer resp.Body.Close()   |   yes    |   yes   |  yes  |
| handle gzip encoding      |   yes    |   yes   |  NO   |
| limit io.Reader           |   yes    |   yes   |  yes  |
| netxlite.ReadAllContext() |   yes    |   yes   |  yes  |
| handle truncated body     |   yes    |   yes   |  NO   |
| log response body         |   yes    |   yes   |  yes  |
| handle non-200 response   | ️  yes    |   yes*  |  yes* |
| unmarshal JSON            |   yes    |   yes   |  yes  |

The `yes*` means that `httpapi` rejects responses with status codes `>= 400` (like cURL)
while the new package only accepts status codes `== 200`. This difference should be of little
practical significance for all the APIs we invoke and the new behavior is stricter.

The `yes~` means that `httpapi` already receives a marshaled body from a higher-level API
that is part of the same package, while in this package we marshal in `PostJSON`.

## Nil Safety

Consider the following code snippet:

```Go
resp, err := httpclientx.GetJSON[*APIResponse](ctx, epnt, config)
runtimex.Assert((resp == nil && err != nil) || (resp != nil && err == nil), "ouch")
```

Now, consider the case where `URL` refers to a server that returns `null` as the JSON
answer, rather than returning a JSON object. The `encoding/json` package will accept the
`null` value and unmarshal it into a `nil` pointer. So, `GetJSON` will return `nil` and
`nil`, and the `runtimex.Assert` will fail.

The `httpx` package did not have this issue because the usage pattern was:

```Go
var resp APIResponse
err := apiClient.GetJSON(ctx, "/foobar", &resp) // where apiClient implements httpx.APIClient
```

In such a case, the `null` would have no effect and `resp` would be an empty response.

However, it is still handy to return a value and an error, and it is the most commonly used
pattern in Go and, as a result, in OONI Probe. So, what do we do?

Well, here's the strategy:

1. When sending pointers, slices, or maps in `PostJSON`, we return `ErrIsNil` if the pointer,
slice, or map is `nil`, to avoid sending literal `null` to servers.

2. `GetJSON`, `GetXML`, and `PostJSON` include checks after unmarshaling so that, if the API response
type is a slice, pointer, or map, and it is `nil`, we also return `ErrIsNil`.

Strictly speaking, it is still unclear to us whether this could happen with `GetXML` but we have
decided to implements these checks for `GetXML` as well, just in case.

## Refactoring Plan

The overall goal is to replace usages of `httpapi` and `httpx` with usages of `httpclient`.

The following packages use `httpapi`:

1. `internal/experiment/webconnectivity`: uses `httpapi.SeqCaller` to chain calls to all
the available test helpers, which we can replace with using `*Overlapped`;

2. `internal/experiment/webconnectivitylte`: same as above;

3. `internal/ooapi`: uses `httpapi` to define the OONI backend APIs for the purpose of
generating and cross-validating swaggers, which is something we defined as a non-goal given
that we never really managed to do it reliably, and it has only led to code complexity;

4. `internal/probeservices`: uses `httpapi` to implement check-in and the main reason why
this is the case is because it supports `"gzip"` encoding;

The following packages use `httpx`:

1. `internal/cmd/apitool`: this is just a byproduct of `probeservices.Client` embedding
the `httpx.APIClientTemplate`, so this should really be easy to get rid of;

2. `internal/enginelocate`: we're using `httpx` convenience functions to figure out
the probe IP and we can easily replace these calls with `httpclientx`;

3. `internal/oonirun`: uses `httpx` to fetch descriptors and can be easily replaced;

4. `internal/probeservices`: uses `httpx` for most other calls.

Based on the above information, it seems the easiest way to proceed is this:

1. `internal/enginelocate`: replace `httpx` with `httpclientx`;

2. `internal/oonirun`: replace `httpx` with `httpclientx`;

3. `internal/probeservices`: replace the check-in implementation to use `httpclientx`
instead of using the `httpapi` package;

4. `internal/experiment/webconnectivity{,lte}`: replace the `httpapi.SeqCaller` usage
with invocations of `*Overlapped`;

5. remove the `httpapi` and `ooapi` packages, now unused;

6. finish replacing `httpx` with `httpclientx` in `internal/probeservices`

7. remove the `httpx` package.

## Limitations and Future Work

The current implementation of `*Overlapped` may cause us to do more work than needed in
case the network is really slow and an attempt is slowly fetching the body. In such a case,
starting a new attempt duplicates work. Handling this case does not seem straightforward
currently, therefore, we will focus on this as part of future work.

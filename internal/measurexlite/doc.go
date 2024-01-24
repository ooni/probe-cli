// Package measurexlite contains measurement extensions. This package is named "measurex lite"
// because it implements a lightweight approach compared to a previous package named "measurex".
//
// [measurexlite] implements the [dd-003-step-by-step.md] design document. The fundamental data type
// is the [*Trace], which saves events in buffered channels. The [NewTrace] constructor creates
// channels with sufficient capacity for tracing all the events we expect to see for a single use
// connection or for a DNS round trip. If you are not draining the channels, the [*Trace] will
// eventually stop collecting events, though.
//
// As mentioned above, the expectation is that a [*Trace] will only trace a single use connection or
// a DNS round trip. Typically, you create a distinct trace for each TCP-TLS-HTTP or TCP-HTTP or
// QUIC-HTTP or DNS-lookup-with-getaddrinfo or DNS-lookup-with-UDP sequence of operations. There is
// a "trace ID" for each trace, which you provide to [NewTrace]. This ID is copied into the
// "transaction_id" field of the archival network events. Therefore, by using distinct trace IDs
// for distinct operations, you enable [ooni/data] to group related events together.
//
// The [*Trace] features methods that mirror existing [netxlite] methods but implement support for
// collecting network events using the [*Trace]. For example, [*Trace.NewStdlibResolver] is like
// [netxlite.Netx.NewStdlibResolver] but the DNS lookups performed with the resolved returned by
// [*Trace.NewStdlibResolver] generate events that you can collect using the [*Trace].
//
// As mentioned above, internally, the [*Trace] uses buffered channels on which the underlying
// network objects attempt to write when there is an interesting event. As a user of the
// [measurexlite] package, you have methods to extract the events from the [*Trace] channels,
// such as, for example:
//
// - [*Trace.DNSLookupsFromRoundTrip]
//
// - [*Trace.NetworkEvents]
//
// - [*Trace.TCPConnects]
//
// - [*Trace.QUICHandshakes]
//
// - [*Trace.TLSHandshakes]
//
// These methods already return data structures using the archival data format implemented
// by the [model] package and specified in the [ooni/spec] repository. Hence, these structures
// are ready to be added to OONI measurements.
//
// [dd-003-step-by-step.md]: https://github.com/ooni/probe-cli/blob/master/docs/design/dd-003-step-by-step.md
// [ooni/data]: https://github.com/ooni/data
// [ooni/spec]: https://github.com/ooni/spec
package measurexlite

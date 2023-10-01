// Package pdsl contains a parallel DSL for writing OONI experiments.
//
// This package encourages you to create measurement pipelines consisting of several stages.
//
// There are two kind of stages: [Generator] and [Filter]. Both [Generator] and [Filter]
// return in output a channel from which you can stream the values they produce. Both will
// close the channel to indicate that they have finished producing values. However, while
// a [Generator] takes in input a value, a [Filter] takes in input a channel produced by
// another [Filter] or a [Generator]. A typical pipeline starts with a [Generator]
// after which you typically include one or more [Filter].
//
// Most [Generator] and [Filter] perform network operations, such as using
// the DNS to resolve domain names, or establishing a TCP connection. Because
// these operations may fail, most [Generator] and [Filter] do not deal with
// bare values, rather with [Result]. A [Result] encapsulates either a
// value or an error. A [Generator] or [Filter] that deals with [Result]
// wrapped values MUST check whether a value is an error first. In that
// case, it SHOULD produce a new [Result] wrapped error. Otherwise, it should
// execute the operation and produce a [Result] wrapped error or value. The
// [NewResultError] constructor creates a [Result] wrapping an error while
// the [NewResultValue] constructor creates a [Result] wrapping a value.
//
// The [Merge] operation allows to merge the results of several channels. The
// [Fork] operation allows to excute a [Filter] using a pool of goroutines. By
// combining [Fork] and [Merge], you have parallel pipeline stages.
//
// Pipelines execute in the context of:
//
// - a [context.Context];
//
// - a [Runtime].
//
// Pipeline operations will always drain their input channels and return
// values using their output channels. When the [context.Context] is canceled
// or its timeout expires, the network operations performed by the pipeline
// stage will fail and cause the proper errors to be returned.
//
// The [Runtime] automatically tracks all the network connections and
// other resources created by pipeline stages. You MUST call the [Runtime]
// Close method when done to ensure you close those tracked connections.
//
// Specific pipeline stages will ask the [Runtime] to create a [Trace] when
// it makes sense to start collecing a set of related OONI observations. However,
// the [NewMinimalRuntime] constructor creates the most minimal [Runtime] that
// does not collect any OONI observation. You SHOULD use this constructor if you
// are not interested in collecting OONI observations.
package pdsl

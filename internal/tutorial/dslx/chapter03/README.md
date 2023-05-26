
# Chapter 3: Parallelizing `dslx` measurements

In the last chapter we have learned how to implement a complete OONI Probe experiment using dslx.
This chapter extends our dslx toolbox by introducing parallel measurements.

If you want, you can try integrating the following parallel (pseudo)code into the SNI blocking
experiment we implemented last chapter.

## `Map`: Single pipeline, multiple input

`dslx` provides us with the `Map` functionality which we can use to apply a pipeline
over multiple inputs in parallel.

`Map` receives the measurement context, the number of goroutines to use,
the pipeline to be applied, and the input values as a channel.

Since we likely have created the input endpoints as a slice instead of a channel,
we use `dslx.StreamList` which takes values from a slice and puts them in a
channel to be consumed by `Map`.

```Go
// pseudo code
resultChannel := dslx.Map(
		context.Background(),
		dslx.Parallelism(2), // number of goroutines to use
		pipeline,
		dslx.StreamList(endpoints...), // create channel for streaming a list of endpoints
)
```

The above pseudo-code applies the pipline to multiple endpoints.

## `ParallelAsync`: Multiple pipelines, single input

We can use `ParallelAsync` to apply multiple different pipelines to the same input:

`ParallelAsync` receives the measurement context, the number of goroutines to use,
the input endpoint, and the pipelines (i.e. composed `dslx` functions) as a channel.

We use `dslx.StreamList` which takes the pipeline functions from a slice and puts
them in a channel to be consumed by `ParallelAsync`.

```Go
// pseudo code
resultChannel := dslx.ParallelAsync(
	context.Background(),
	dslx.Parallelism(2), // number of goroutines to use
	inputDomain,
	dslx.StreamList(
		dslx.DNSLookupGetaddrinfo(),
		dslx.DNSLookupUDP("8.8.8.8:53"),
	),
)
```

The above pseudo code runs a parallel DNS lookup using two distinct resolvers: the
system resolver and a DNS-over-UDP resolver using `8.8.8.8:53`.


## Handling async output

`Map` and `ParallelAsync` return a channel that we can drain into a slice using
`Collect` to receive the results as a list of type `[]*dslx.Maybe`:

```Go
results := dslx.Collect(resultChannel)
```

Since there is now a list of multiple results of type `dslx.Maybe`,
and each `dslx.Maybe` has a separate error, we use the utility function
`dslx.FirstError` to obtain the first occurred error.

```Go
_, err := dslx.FirstError(results...)
```

We can merge all observations of the multiple results into a single slice by
using `dslx.ExtractObservations`:

 ```Go
allObservations := dslx.ExtractObservations(results...)
```

## Approaches to parallelize SNI blocking

`Map` could be used to parallelize the SNI blocking experiment if we decide to measure
all endpoints of the testhelper. (In the implementation in chapter02, we only considered
the first resolved IP endpoint.)

Using `Map` we can run the measurement pipeline in parallel over multiple endpoints:

```Go
resultChannel := dslx.Map(
	ctx,
	dslx.Parallelism(2),
	pipelineTarget,
	dslx.StreamList(endpoints...),
)
```

In contrast, `ParallelAsync` could be used to parallelize the target and control measurement of
the SNI blocking experiment, over the same endpoint:

```Go
resultChannel := dslx.ParallelAsync(
	ctx,
	dslx.Parallelism(2),
	endpoint,
	dslx.StreamList(pipelineTarget, pipelineControl),
)
```

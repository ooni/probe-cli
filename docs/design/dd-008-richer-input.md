# Richer Input

|              |                                                |
|--------------|------------------------------------------------|
| Author       | [@bassosimone](https://github.com/bassosimone) |
| Last-Updated | 2024-06-04                                     |
| Reviewed-by  | N/A                                            |
| Status       | work in progress                               |

## Problem Statement

Traditionally, OONI experiments measure _inputs_. For example, the command in
Listing 1 measures the `https://www.example.com/` URL using the `web_connectivity` experiment.

```
./miniooni web_connectivity -i https://www.example.com/
```

**Listing 1** Running Web Connectivity with a given URL using `miniooni`.

Some experiments support providing _options_ via command line. For example, the command
in Listing 2 runs the `dnscheck` experiment measuring `https://8.8.8.8/dns-query` and using
the `HTTP3Enabled` option set to `true`.

```
./miniooni dnscheck -i https://8.8.8.8/dns-query -O HTTP3Enabled=true
```

**Listing 2** Running DNSCheck with a given URL and options using `miniooni`.


Additionally, OONI Run v2 allows to express running experiments with options. For
example, the JSON document in Listing 3 is equivalent to the code in Listing 2.

```JSON
{
	"nettests": [{
		"test_name": "dnscheck",
		"options": {
			"HTTP3Enabled": true
		},
		"inputs": [
			"https://8.8.8.8/dns-query"
		]
	}]
}
```

**Listing 3** Running DNSCheck with a given URL and options using an OONI Run v2 descriptor.

However, when OONI Probe runs without options, as shown in Listing 4 and
Listing 5, it either uses hardcoded defaults or OONI backend APIs to retrieve the
list of *inputs* to measure, without specifying additional options.

```
./miniooni dnscheck
```

**Listing 4** Running DNSCheck without URLs and inputs using `miniooni`.

```
ooniprobe run experimental
```

**Listing 5** Running DNSCheck indirectly through the `experimental` suite using `ooniprobe`.

Yet, the possibility of specifying options through the OONI backend is important to
widen the kind of network experiments we can perform and can inform decisions such as
customising the measurement algorithm and detect throttling cases.

Therefore, there is a need to remove the codebase bottlenecks preventing OONI Probe
from measuring tuples consisting of *inputs* and *options*.

We define *richer input* the combination of specific *inputs* and *options* and we say
that removing these bottlenecks *is* making richer input possible.

The rest of this document explains how we're going to do this.

## Problem Analysis

As of [v3.22.0](https://github.com/ooni/probe-cli/tree/v3.22.0) OONI Probe is
written as a framework for executing experiments. The framework handles:

1. discovering the probe location;

2. communicating with the backend for loading inputs;

3. opening reports and submitting measurements.

That is, experiments plug into the framework and most of the job
around executing them is performed by the framework itself.

Historically, this design originates from the [legacy Python implementation]
(https://github.com/ooni/probe-legacy) and the [Measurement Kit engine](
https://github.com/measurement-kit/measurement-kit). The current codebase, in
particular, was originally written to use Measurement Kit as its measurement
engine and only years later we rewrote the engine in Go.

This framework oriented design has its benefits and is what allowed us to
switch from the Measurement Kit engine to the current engine.

However, as far as richer input is concerned, this design also comes with its
own set of drawbacks. In particular, with richer input we would like to
build open the example set by the [tor experiment](
https://github.com/ooni/spec/blob/master/nettests/ts-023-tor.md), which invokes
its own backend API to obtain experiment-specific inputs and options.

However, fitting the kind of richer input required by each experiment into
the framework would put excessive levels of pressure into the framework and
will increase the maintenance efforts.

## Proposed Solution Overview

We propose to refactor how the framework runs experiments so that:

1. There is a new entry point for invoking experiments that is responsible
of gathering (possibly richer) input for the experiment and cyclying through
such input to produce one or more measurements.

2. There is a library of functions that supports experiments in doing so.

By implementing this transformation, each experiment will be able to evolve
independently in terms of its richer input requirements.

The rest of this document elaborates on how to practically do this in
terms of the existing OONI Probe codebase.

## Existing Framework

Let us strictly focus on creating experiments, loading input for
experiments, and measuring each input, thus exclucing other framework
functionality that are out of the scope of this document.

With this simplifying assumption, running an experiment entails
the operations described in pseudo-Go by Listing 6.

```Go
// Create a new measurement session using the given config.
//
// The sessionConfig field type is [engine.SessionConfig] and contains
// configuration such as the proxy or the directory paths to use.
//
// The type of a session is [*engine.Session].
sess, err := engine.NewSession(ctx, sessionConfig)
runtimex.PanicOnError(err)

// Create a new experiment builder given the experiment name.
//
// The type of an experiment builder is [model.ExperimentBuilder].
builder, err := sess.NewExperimentBuilder(experimentName)
runtimex.PanicOnError(err)

// Set options set on the command line. The env.Options field here
// is a map[string]any where any is typically a scalar.
//
// When using miniooni, the env.Options come from the CLI.
//
// When using OONI Run v2, the env.Options come from the descriptor.
//
// OONI Probe CLI, desktop, and mobile usually does not allow setting
// experiment options, so this field would be empty.
for key, value := range env.Options {
	err := builder.SetOptionAny(key, value)
	runtimex.PanicOnError(err)
}

// Create an input loader for loading input given the specific
// experiment indicated by its experiment builder.
//
// Both env.Inputs and env.InputFiles are []string values.
//
// When using miniooni or ooniprobe, those values are specified using
// the command line for experiments supporting this feature.
//
// When using desktop or mobile, one can use the GUI to configure
// which are the inputs for experiments supporting this feature.
//
// When using OONI Run v2, inputs are specified by the JSON descriptor.
//
// The type of an input loader is [*engine.InputLoader].
loader := &engine.InputLoader{
	CheckInConfig: /* ... */,
	ExperimentName: experimentName,
	InputPolicy: builder.InputPolicy(),
	Logger: sess.Logger(),
	Session: sess,
	StaticInputs: env.Inputs,
	SourceFiles: env.InputFiles,
}

// Actually load inputs. The return value is a []model.OOAPIURLInfo
// that corresponds to the following data structure:
//
//	type OOAPIURLInfo struct {
//		CategoryCode string
//		CountryCode  string
//		URL          string
//	}
//
// As you can see, this structure is very specifically taylored to Web
// Connectivity and isn't suitable to represent richer input.
//
// So, while we can set options from the command line, as shown above, we
// don't have the structure to provide options from the backend API.
//
// Note that experiments that do not take any input, such as, for example,
// ndt and dash, use an InputPolicy that returns an empty string. So,
// we always end up running the experiment at least once in the loop below.
inputs, err := loader.Load(ctx)
runtimex.PanicOnError(err)

// Create an experiment from the builder.
//
// The type of an experiment is [model.Experiment].
experiment := builder.NewExperiment()

// Measure each input. (See above note regarding experiments without input.)
for _, input := range inputs {
	// Measure the given input with the options implicitly configured
	// when invoking the builder.SetOptionAny method.
	//
	// The type of meas is [*model.Measurement].
	meas, err := experiment.MeasureWithContext(ctx, input)

	// On error we continue running for other inputs. Note that an error
	// means, by convention, that something went really bad while running
	// the experiment (such as, the input having the wrong format). Any
	// censorship condition SHOULD NOT cause exp.Measure to return an error.
	if err != nil {
		continue
	}

	// Do something with the measurement.
	// ...
}
```

**Listing 6** Simplified set of operations required for running an experiment
from the point of view of `miniooni`, OONI Probe CLI, and mobile. We are omitting
details such as opening a report, submitting measurements, updating the DB.

The above listing (and generally the codebase) is slightly more complex than it
should. In particular, `model.ExperimentBuilder` and `model.Experiment` are
always and only implemented by the concrete types `*engine.experimentBuilder`
and `*engine.experiment`. (In any case, it makes sense for the interfaces to
exist for testability, however, it would be more idiomatic for the actually
returned types to be concrete types rather than interfaces.)

The `*engine.experimentBuilder` is implemented in terms of the `*registry.Factory`
type, which describes how to construct an experiment.

The `*engine.experiment` wraps a `model.ExperimentMeasurer`. Each implementation
of an OONI experiment contains, as its most top level functionality, a factory
function to create a `model.ExperimentMeasurer` instance constructed by the `registry`
package when the framework is creating an experiment instance.

## Proposed Solution Details

We aim to proceed with incremental refactoring. As such, rather than changing
the definition of an experiment, we aim to create a new definition that supports
richer input, such that we can incrementally implement richer input. At the end
of this process, there will be a large cleanup removing unused legacy code.



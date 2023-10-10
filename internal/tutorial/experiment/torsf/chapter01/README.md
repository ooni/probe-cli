
# Chapter I: main.go using the real torsf implementation

In this chapter we will write together a `main.go` file that
uses the real `torsf` implementation to run the experiment.

(This file is auto-generated from the corresponding source file,
so make sure you don't edit it manually.)

## The torsf experiment

This experiment attempts to bootstrap the `tor` binary using
Snowflake as the pluggable transport.

You can read the [specification](https://github.com/ooni/spec/blob/master/nettests/ts-030-torsf.md)
of the `torsf` experiment in the [ooni/spec](https://github.com/ooni/spec)
repository. (The `ooni/spec` repository is the repository
containing the specification of all OONI nettests, as well
as of the data formats used by OONI.)

## The main.go file

We define `main.go` file using `package main`.

```Go
package main

```

### Imports

Then we add the required imports.

```Go
import (
```

These are standard library imports.

```Go
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"

```

The apex/log library is the logging library used by OONI Probe.

```Go
	"github.com/apex/log"

```

The torsf package contains the implementation of the torsf experiment.

```Go
	"github.com/ooni/probe-cli/v3/internal/experiment/torsf"

```

The mockable package contains widely used mocks.

```Go
	"github.com/ooni/probe-cli/v3/internal/legacy/mockable"

```

The model package contains the data model used by OONI experiments.

```Go
	"github.com/ooni/probe-cli/v3/internal/model"

```

We will need the execabs library to check whether there is
a binary called `tor` in the `PATH`.

```Go
	"golang.org/x/sys/execabs"
)

```

### Main function

Finally, here's the code of the `main function`.

```Go
func main() {
```

We start by checking whether there is an executable named `"tor"` in
the `PATH`. If there is no such executable, we fail with an error.

```Go
	if _, err := execabs.LookPath("tor"); err != nil {
		log.Fatal("cannot find the tor executable in path")
	}
```

Then, we create a temporary directory to hold any state that may be
required either by the `tor` or by the Snowflake pluggable transport.

```Go
	tempdir, err := ioutil.TempDir("", "")
	if err != nil {
		log.WithError(err).Fatal("cannot create temporary directory")
	}
```

### Creating the experiment measurer

All OONI experiments implement a function called
`NewExprimentMeasurer` that allows you to make
an `ExperimentMeasurer` instance. The `ExperimentMeasurer`
is an `interface` defined by the `model` package we
imported above. Because we don't want to configure
any setting (and the experiment does not support any
setting anyway), here we're passing to the
`NewExperimentMeasurer` factory an empty `Config`.

```Go
	m := torsf.NewExperimentMeasurer(torsf.Config{})
```

### Creating the measurement

Next, we create an empty `Measurement`. OONI measurements
are JSON data structures that contain generic fields common
to all OONI experiments and experiment-specific data. The
experiment-specific data is contained by a the `test_keys`
field of the `Measurement`.

In the *real* OONI implementation, there is common code
that fills the several fields of a `Measurement`. For
example, it will fill the country code and the autonomous
system number of the network in which the OONI Probe is
running. Because this is just an example to illustrate
how to write experiments, we will not bother with doing
that. Instead, we will pass to the experiment just an
emtpy measurement where no field has been set.

```Go
	measurement := &model.Measurement{}
```

### Creating the callbacks

Then, we create an instance of the experiment callbacks. The
experiment callbacks historically groups a set of callbacks
called when the measurer is running. At the moment of writing
this note, the `model.ExperimentCallbacks` contains just a
single method called `OnDataUsage`, which is used to tell the
caller which is the amount of data used by the experiment.

Because this is an example for illustrative purposes, here
we construct an implementation of `ExperimentCallbacks` that
just prints the data usage using the `log.Log` logger.

```Go
	callbacks := model.NewPrinterCallbacks(log.Log)
```

### Creating a session

The `ExperimentMeasurer` also wants a `Session`. In normal
OONI code, the `Session` is a data structure containing
information regarding the current measurement session. Since
this is just an illustrative example, rather than creating
a real `Session` instance, we use much-simpler mock.

The interface required by a `Session` is called
`ExperimentSession` and is part of the `model` package.

Here we configure this mockable session to use `log.Log`
as a logger and the previously computed temp dir.

```Go
	sess := &mockable.Session{
		MockableLogger:  log.Log,
		MockableTempDir: tempdir,
	}
```

# Running the experiment

At last, it's time to run the experiment using all the
previously constructed data structures. The `Run` function
is the main function you need to implement when you are
defining a new OONI experiment.

By convention, the `Run` function only returns an error
when some precondition required by the experiment is
not met. Say that, for example, the experiment needs a
port listening on the local host. If we cannot create
such a port, we will return an error to the caller.

For network errors, instead, we return nil. Consider the
case where we connect to a remote host and the connection
fails. This is not really an error, rather it's a result
that we will include into the measurement.

Apart from the other arguments that we discussed previously,
the `Run` function also wants a `context.Context` as its
first argument. The context is used to interrupt long running
functions early, and our code (mostly) honours contexts.

Since here we are just writing a simple example, we don't
need any fancy context and we pass a `context.Background` to `Run`.

```Go
	ctx := context.Background()
	args := &model.ExperimentArgs{
		Callbacks:   callbacks,
		Measurement: measurement,
		Session:     sess,
	}
	if err = m.Run(ctx, args); err != nil {
		log.WithError(err).Fatal("torsf experiment failed")
	}
```

### Printing the measurement result

The `Run` function modifies the `TestKeys` (`test_keys` in JSON)
field of the measurement. The real OONI implementation would
now submit this measurement. Because this is an illustrative example,
we will just pretty-print the measurement on the `stdout`.

```Go
	data, err := json.Marshal(measurement)
	if err != nil {
		log.WithError(err).Fatal("json.Marshal failed")
	}
	fmt.Printf("%s\n", data)
}

```

## Running the code

You can now run this code as follows:

```
$ go run ./experiment/torsf/chapter01 | tail -n 1 | jq
[snip]
{
  "data_format_version": "",
  "input": null,
  "measurement_start_time": "",
  "probe_asn": "",
  "probe_cc": "",
  "probe_network_name": "",
  "report_id": "",
  "resolver_asn": "",
  "resolver_ip": "",
  "resolver_network_name": "",
  "software_name": "",
  "software_version": "",
  "test_keys": {
    "bootstrap_time": 68.909067459,
    "failure": null
  },
  "test_name": "",
  "test_runtime": 0,
  "test_start_time": "",
  "test_version": ""
}
```

We have snipped through logs and we have used `jq` to
pretty print the measurement. You see that all the fields
except the `test_keys` are empty.

Let us now analyze the content of the `test_keys`:

- the `bootstrap_time` field contains the time (in seconds) to
bootstrap `tor` using the Snowflake transport;

- the `failure` field contains the error that occurred, if
any, or `null` if no error occurred.

## Concluding remarks

This is all you need to know in terms of minimal code for
running an OONI experiment. In the remainder of this tutorial,
we will show how to reimplement the `torsf` experiment.

Apart from minor changes, the `main.go` file would basically
not change for the remainder of this tutorial.

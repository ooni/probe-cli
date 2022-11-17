
# Chapter II: creating an empty experiment

In this chapter we will create an empty experiment and replace
the code calling the real `torsf` experiment in `main.go` to
call our empty experiment instead.

(This file is auto-generated from the corresponding source file,
so make sure you don't edit it manually.)

## Changes in main.go

In `main.go` we will simply replace the call to the
`torsf.NewExperimentMeasurer` function with a call to
a `NewExperimentMeasurer` function that we are going
to implement as part of this chapter.

After you do this, you also need to remove the now-unneded
import of the `torsf` package.

There are no additional changes to `main.go`.

```Go
	m := NewExperimentMeasurer(Config{})
```

## The torsf.go file

This file will contain the implementation of the
`NewExperimentMeasurer` function.

As usual we start with the `package` declaration and
with the few imports we need to add.

```Go
package main

import (
	"context"
	"time"

	"github.com/ooni/probe-cli/v3/internal/model"
)

```

### Data structures

Next, we define data structures.

Config contains config for the torsf experiment. As for the real
`torsf` experiment, we don't have any specific config, so we keep
the structure empty. We still need to define a `Config` struct
here, because, by convention, all OONI experiments have a `Config`.

```Go
type Config struct{}

```

Measurer is the torsf measurer. This structure implements the
`model.ExperimentMeasurer` interface, as we will see below.

Most OONI experiments have a measurer that contains as its unique
field the specific configuration. Here we do the same.

```Go
type Measurer struct {
	config Config
}

```
NewExperimentMeasurer creates a new model.ExperimentMeasurer
instance for performing `torsf` measurements. This function
will just assemble a new instance of `Measurer` with the `config`
that was passed as an argument.

```Go
func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

```

### Implementing the model.ExperimentMeasurer.

Now it's time to implement the methods required by the `model`'s
`ExperimentMeasurer` interface.

ExperimentName implements ExperimentMeasurer.ExperimentName. This function
returns the name of the experiment. This code is used by generic code
manipulating the experiment to print the experiment name.

```Go
func (m *Measurer) ExperimentName() string {
	return "torsf"
}

```

ExperimentVersion implements ExperimentMeasurer.ExperimentVersion. This
function returns the version of the experiment. This code is also used by
generic code manipulating the experiment to print the experiment version.

```Go
func (m *Measurer) ExperimentVersion() string {
	return "0.1.0"
}

```

Run implements ExperimentMeasurer.Run. This is the most interesting
function, where we run the experiment proper. In the previous chapter
we learned how to call this function from a `main.go` file. Here,
instead, we're going to create a minimal stub. In the subsequent
chapters, finally, we will modify this function until it is a
minimal implementation of the `torsf` experiment.

```Go
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	_ = args.Callbacks
	_ = args.Measurement
	sess := args.Session
```
As you can see, this is just a stub implementation that sleeps
for one second and prints a logging message.

```Go
	time.Sleep(time.Second)
	sess.Logger().Info("hello from the torsf experiment!")
	return nil
}

```
### Summary keys

Before concluding this chapter, we also need to create the `SummaryKeys`
for this experiment. For historical reasons, the `TestKeys` of each
experiment is an `interface{}`. Every experiment also defines a `SummaryKeys`
data structure and a `GetSummaryKeys` method to convert the opaque
result of a measurement to the summary for such an experiment.

The experiment summary is *only* used by the OONI Probe CLI.

SummaryKeys contains summary keys for this experiment. Because this is
just an illustrative tutorial, we will just include a single key, named
`IsAnomaly`. This key is not exported as JSON and is used by the OONI
Probe CLI to inform the user of whether this measurement is ordinary or
anomalous. All OONI experiments' `SummaryKeys` contain such a field.

```Go
type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

```

GetSummaryKeys implements model.ExperimentMeasurer.GetSummaryKeys. This
method just converts the `TestKeys` inside `measurement` to an instance of
the `SummaryKeys` structure. For now, we'll just implement a stub returning
fake `SummaryKeys` declaring there was no anomaly.

```Go
func (m *Measurer) GetSummaryKeys(measurement *model.Measurement) (interface{}, error) {
	return &SummaryKeys{IsAnomaly: false}, nil
}

```

## Running the code

We can run the code written in this chapter as follows:

```
$ go run ./experiment/torsf/chapter02
2021/06/21 20:48:32  info hello from the torsf experiment!
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
  "test_keys": null,
  "test_name": "",
  "test_runtime": 0,
  "test_start_time": "",
  "test_version": ""
}
```

Here you see that we're printing the log message and
that the `test_keys` are `null`.

The OONI data processing popeline will not be so happy
if we pass it a `null` settings, because there is not
much interesting data in there. We will thus start filling
it in the next chapter.

# Richer Input

|              |                                                |
|--------------|------------------------------------------------|
| Author       | [@bassosimone](https://github.com/bassosimone) |
| Last-Updated | 2024-06-07                                     |
| Reviewed-by  | N/A                                            |
| Status       | living document                                |

This document is a living document documenting our in-progress design
for implementing [richer input](https://github.com/ooni/ooni.org/issues/1291).

We define as richer input the possibility of using the OONI backend API to
provide OONI experiments with not only inputs but also options.

## Problem Statement

Traditionally, OONI experiments measure _inputs_. For example, the following
command measures the `https://www.example.com/` URL using the
`web_connectivity` experiment.

```bash
./miniooni web_connectivity -i https://www.example.com/
```

Some experiments support providing _options_ via command line. For example,
the following command runs the `dnscheck` experiment measuring
`https://8.8.8.8/dns-query` and using the `HTTP3Enabled` option set to `true`.

```bash
./miniooni dnscheck -i https://8.8.8.8/dns-query -O HTTP3Enabled=true
```


Additionally, OONI Run v2 allows to run experiments with options. For example,
the following JSON document is equivalent to the previous `miniooni` command:

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

However, when OONI Probe runs without options, as shown in the following
commands, it either uses hardcoded defaults or OONI backend APIs to retrieve
the list of *inputs* to measure. Crucially, this list of inputs comes with
no additional options.

```bash
# Running dnscheck without options with miniooni
./miniooni dnscheck

# Same as above but with ooniprobe
ooniprobe run experimental
```

To better understand what is going on, we need to briefly take a look at the types
and interfaces used by [OONI Probe v3.22.0](https://github.com/ooni/probe-cli/tree/v3.22.0), which are roughly the following:

```Go
// ExperimentDescriptor is filled by OONI Run or by the command line.
type ExperimentDescriptor struct {
	Inputs   []string        // OPTIONAL
	Options  map[string]any  // OPTIONAL
	Name     string          // MANDATORY
}

// ExperimentBuilder is the constructor for an experiment.
type ExperimentBuilder interface {
	// SetOptionsAny configures the options from the descriptor.
	SetOptionsAny(options map[string]any) error

	// NewExperiment constructs an experiment with the options.
	NewExperiment() Experiment
}

// InputLoader loads inputs, typically invoking OONI backend APIs though
// for some experiments the input is hardcoded in the source code.
type InputLoader interface {
	Load(ctx context.Context) ([]OOAPIURLInfo, error)
}

// OOAPIURLInfo is the structure returned by the input loader.
type OOAPIURLInfo struct {
	CategoryCode string
	CountryCode  string
	URL          string
}

// Experiment transforms inputs and options into OONI measurements.
type Experiment interface {
	MeasureWithContext(ctx context.Context, input string) (*Measurement, error)
}
```

With this data model we run experiments using the following pseudo-code:

```Go
// RunExperiment runs the experiment described by the descriptor.
//
// We omit error handling for simplicity.
func RunExperiment(ctx context.Context, descriptor *ExperimentDescritor) {
	// Create experiment builder by name.
	builder, err := NewExperimentBuilder(descriptor.Name)
	// [...]

	// Create empty inputs.
	inputs := []OOAPIURLInfo{}

	// If we have either options or inputs from the user or from OONI RUN v2
	// use them, otherwise load inputs from the OONI backend.
	if len(descriptor.Options) > 0 || len(descriptor.Inputs) > 0 {
		builder.SetOptionsAny(descriptor.Options)
		inputs := NewOOAPIURLInfo(descriptor.Inputs...)
	} else {
		loader := NewInputLoader(builder)
		inputs, err = loader.Load(ctx)
		// [...]
	}

	// Create experiment.
	//
	// Note that this constructor communicates the options to the experiment.
	exp := builder.NewExperiment()

	// Measure inputs
	for _, input := range input {
		meas, err := exp.MeasureWithContext(ctx, input)
		// [...]
	}
}
```

This pseudo-code should clarify the problem. The data structure representing
input (`OOAPIURLInfo`) does not allow loading options from the backend when we
are using an `InputLoader`. We say that adding support for returning options
along with inputs provides us with "richer input", because we will enrich the
input URLs to measure with additional options.

Solving this problem is crucial because most OONI measurements run automatically
in the background with input provided by the backend. Therefore, by enabling
richer input, we open up the possibility of answering specific research questions
requiring options at scale. For example, richer input would enable us to
study [DNS over HTTP/3 is blocking](https://github.com/ooni/probe/issues/2675))
at scale.

## Towards implementing richer input

## Design choice: deprecating the check-in API

We originally envisioned distributing richer input through the check-in API
but we later realized that this design would be problematic because:

1. it prevents us from having experiments implemented as scripts, a solution
that we heavily explored while researching richer input;

2. the check-in API serves URLs for Web Connectivity, which is the most
important experiment we run, which means that changing the component serving
the richer input API requires careful vetting of the changes and could
potentially hamper our ability to iterate quickly.

For this reason, we determined that all richer input enabled experiments
will eventually invoke their own API, like the `tor` experiment does.

## Design choice: distributing feature flags using check-in

OONI Probe consists of several experiments, some of which are stable, such
as Web Connectivity, and some of which are hightly experimental, such as the
recently added `openvpn` experiment.

So, we need a mechanism to flag experiments as unstable and remotely
enable/disable them if needed. Because we implemented this functionality
while still researching richer input, currently we use the check-in API
feature flags to implement this functionality.

We initially implemented check-in feature flags to dynamically enable the
experimental Web Connectivity LTE implementation in [probe-cli#1123](
https://github.com/ooni/probe-cli/pull/1123) for selected users.

Subsequently, in [probe-cli#1355](https://github.com/ooni/probe-cli/pull/1355),
we extended the feature flags to conditionally enable/disable the experiments
that we know could potentially become problematic.

## Refactoring: enabling richer input

In [probe-cli#1615](https://github.com/ooni/probe-cli/pull/1615) we modified
the codebase so that, instead of using `OOAPIURLInfo` we now use:

```Go
type ExperimentTarget struct {
	Category() string  // equivalent to OOAPIURLInfo.CategoryCode
	Country() string   // equivalent to OOAPIURLInfo.CountryCode
	Input() string     // equivalent to OOAPIURLInfo.URL
}

type InputLoader = TargetLoader // we renamed to reflect the change of purpose

type TargetLoader interface {
	Load(ctx context.Context) ([]ExperimentTarget, error)
}

type Experiment interface {
	MeasureWithContext(ctx context.Context, target ExperimentTarget) (*Measurement, error)
}
```

Note that we also renamed the `InputLoader` to `TargetLoader` to reflect the fact that
we're not loading bare input anymore, rather we're loading richer input targets.

Also, `OOAPIURLInfo` implements `ExperimentTarget` and the mapping between its fields
and `ExperimentTarget` methods is made explicit by comments in the code above.

In [probe-cli#1617](https://github.com/ooni/probe-cli/pull/1617)
and [probe-cli#1618](https://github.com/ooni/probe-cli/pull/1618)
we additionally modified the `ExperimentBuilder` model as follows:

```Go
type ExperimentBuilder interface {
	// ...

	NewTargetLoader(staticInputs []string) TargetLoader
}
```

Therefore, now we create an `ExperimentBuilder`-dependent `TargetLoader` and
each experiment could use its own implementation, if needed.

Thanks to this change, code in `./cmd/ooniprobe` and `./internal/oonirun` (used by
`./internal/cmd/miniooni` to run experiments) now is written in a style that
supports using richer input. We can therefore update our pseudo-code:

```diff
 // RunExperiment runs the experiment described by the descriptor.
 //
 // We omit error handling for simplicity.
 func RunExperiment(ctx context.Context, descriptor *ExperimentDescritor) {
 	// Create experiment builder by name.
 	builder, err := NewExperimentBuilder(descriptor.Name)
 	// [...]

-	// Create empty inputs.
-	inputs := []OOAPIURLInfo{}

 	// If we have either options or inputs from the user or from OONI RUN v2
 	// use them, otherwise load inputs from the OONI backend.
-	if len(descriptor.Options) > 0 || len(descriptor.Inputs) > 0 {
-		builder.SetOptionsAny(descriptor.Options)
-		inputs := NewOOAPIURLInfo(descriptor.Inputs...)
-	} else {
-		loader := NewInputLoader(builder)
-		inputs, err = loader.Load(ctx)
-		// [...]
-	}
+	builder.SetOptionsAny(descriptor.Options)
+	loader := builder.NewTargetLoader(descriptor.Inputs...)
+	inputs, err := loader.Load(ctx)
+	// [...]

 	// Create experiment.
 	//
 	// Note that this constructor communicates the options to the experiment.
 	exp := builder.NewExperiment()

 	// Measure inputs
 	for _, input := range input {
 		meas, err := exp.MeasureWithContext(ctx, input)
 		// [...]
 	}
 }
```

In turn, the specific implementation of `Load` would do something like:

```Go
type target struct {
	options map[string]any
	input  string
}

var _ ExperimentTarget = &target{}
// [...]

type targetLoader struct {
	// inputs and options is configured by builder.NewTargetLoader
	inputs  []string
	options map[string]any
}

func (tl *targetLoader) Load(ctx context.Context) ([]model.ExperimentTarget, error) {
	if len(descriptor.Options) <= 0 && len(descriptor.Inputs) <= 0 {
		return tl.invokeRicherInputAPI(ctx)
	}
	inputs := NewOOAPIURLInfo(descriptor.Inputs...)
	var output []model.ExperimentTarget
	for _, input := range inputs {
		output = append(outputs, &target{tl.options, input})
	}
	return output, nil
}
```

We also modified richer input enabled experiments (for now just `dnscheck`)
such that, rather than setting the options as part of `builder.NewExperiment`,
we are now passing both options and each input together. In pseudo-code,
the changes roughly look like this:

```diff
 type ExperimentArgs struct {
	// [...]

	Measurement *Measurement

+	Target model.ExperimentTarget
 }

 type ExperimentMeasurer interface {
	Run(ctx context.Context, args *ExperimentArgs) error
 }

 type experimentMeasuer struct{
-	options map[string]any
 }

-func NewMeasurer(options map[string]any) ExperimentMeasurer {
-	return &experimentMeasurer{options}
+func NewMeasurer() ExperimentMeasurer {
+	return &experimentMeasurer{}
 }

 var _ ExperimentMeasurer = &experimentMeasurer{}

 func (mx *experimentMeasurer) Run(ctx context.Context, args *ExperimentArgs) error {
-	input := string(args.Measurement.Input)
-	options := mx.options
+	if args.Target == nil {
+		return ErrInputRequired
+	}
+	input := args.Target.(*target).input
+	options := args.Target.(*target).options
	// [...]
 }
```

## Next steps

This is a rough sequence of next steps that we should expand as we implement
additional bits of richer input and for which we need reference issues.

*  setting `(*Measurement).Options` inside `(*engine.experiment).newMeasurement`
rather than inside the `oonirun` package.

*  fully convert `dnscheck`'s static list to live inside `dnscheck` instead of
`targetloading` and to use the proper richer input.

*  implement backend API

	*  for serving `dnscheck` richer input.

	*  implement backend API for serving `stunreachability` richer input.

	*  implement backend API for serving `openvpn` richer input.

*  deliver feature flags using experiment-specific richer input rather
than using the check-in API (and maybe keep the caching support?).

*  deliver OONI Run v2 options as a `json.RawMessage` rather than passing
through a `map[string]any` such that we can support non-scalar options when
using OONI Run v2.

*  try to eliminate `InputPolicy` and instead have each experiment define
its own constructor for the proper target loader, and split the implementation
inside of the `targetloader` package to have multiple target loaders.

	*  rework `pkg/oonimkall` to invoke a target loader rather than relying
	on the `InputPolicy`

	*  make sure richer-input-enabled experiments can run with `oonimkall`
	after we have performed the previous change

*  devise long term strategy for delivering richer input to `oonimkall`
from mobile apps, which we'll need as soon as we convert the IM experiments



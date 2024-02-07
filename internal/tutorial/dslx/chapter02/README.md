
# Chapter 2: Re-implementing SNI blocking

## The SNI blocking experiment

In this tutorial, we will re-implement the existing [SNI blocking](
https://github.com/ooni/spec/blob/master/nettests/ts-024-sni-blocking.md)
experiment using dslx.

SNI blocking aims to understand whether there is blocking triggered by the content of
the TLS Hello's SNI field. The SNI (Server Name Indication) is a
[TLS extension](https://www.rfc-editor.org/rfc/rfc6066) that reveals the requested
service's domain name. For a given SNI/domain, this nettest talks to an
uncensored test helper server, while using the specified target SNI.

We will implement a *simplified version* of SNI blocking to demonstrate how to use dslx
to write an OONI network experiment. We chose this experiment to re-implement because
it consists of many building blocks that we can all represent using dslx.

SNI blocking receives a `target` SNI, a `control_sni`, and the testhelper's address.
For each `target` SNI, a typical runthrough of SNI blocking looks like this:

* connect to testhelper using `target` as SNI

* connect to testhelper using `control_sni` as SNI


## Getting started

We use `package chapter02`.

```Go
package chapter02

```

### Imports

Then we add the required imports.

```Go
import (
	"context"
	"errors"
	"net"

	"github.com/ooni/probe-cli/v3/internal/model"
	"github.com/ooni/probe-cli/v3/internal/runtimex"
	"github.com/ooni/probe-cli/v3/internal/x/dslx"
)

```

### Test name and version

We give our re-implementation the new name "simple_sni".

```Go
const (
	testName    = "simple_sni"
	testVersion = "0.1.0"
)

```

### Config

Config contains the experiment config.

`ControlSNI` is the SNI to be used for the control. `TestHelperAddress` is the
address of the test helper TCP endpoint (e.g., `1.2.3.4:5678`).

```Go
type Config struct {
	ControlSNI        string
	TestHelperAddress string
}

```

## The output data format

`TestKeys` contains the SNI blocking test keys.

```Go
type TestKeys struct {
	Control Subresult `json:"control"`
	Result  string    `json:"result"`
	Target  Subresult `json:"target"`
}

```

`Subresult` contains the keys of a single measurement
that targets either the target or the control.

```Go
type Subresult struct {
	Failure       *string                                   `json:"failure"`
	NetworkEvents []*model.ArchivalNetworkEvent             `json:"network_events"`
	SNI           string                                    `json:"sni"`
	TCPConnect    []*model.ArchivalTCPConnectResult         `json:"tcp_connect"`
	THAddress     string                                    `json:"th_address"`
	TLSHandshakes []*model.ArchivalTLSOrQUICHandshakeResult `json:"tls_handshakes"`
	Cached        bool                                      `json:"-"`
}

```

## The Measurer

The `Measurer` performs the measurement and implements `ExperimentMeasurer`; i.e., the
interface used by all OONI Probe experiments. The `Measurer` contains the config
and an `atomic.Int64` to generate unique IDs (needed by dslx to assign to each run
of dslx pipelines a unique identifier).

```Go
type Measurer struct {
	config Config
}

var _ model.ExperimentMeasurer = &Measurer{}

```

To fulfill the `model.ExperimentMeasurer` interface, we also need basic implementations
of the following functions. To understand exactly what they do, please refer to
[the torsf tutorial](../../../torsf). Because discussing them would be off-topic
for this tutorial, we are not going to provide additional details.

```Go

func (m *Measurer) ExperimentName() string {
	return testName
}

func (m *Measurer) ExperimentVersion() string {
	return testVersion
}

func NewExperimentMeasurer(config Config) model.ExperimentMeasurer {
	return &Measurer{config: config}
}

type SummaryKeys struct {
	IsAnomaly bool `json:"-"`
}

```

## `Run`: The experiment code

`Run` is required by the `ExperimentMeasurer` interface and contains the measurement code.
So, this is where we will use `dslx` to implement the SNI blocking experiment.

```Go
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
```

`measurement` contains metadata, the (required) input in form of
the target SNI, and the nettest results (`TestKeys`).

```Go
	measurement := args.Measurement
	if measurement.Input == "" {
		return errors.New("experiment requires measurement.Input")
	}
	targetSNI := string(measurement.Input)

```

We create a new instance of `TestKeys` to store the results.

```Go
	tk := &TestKeys{}
	measurement.TestKeys = tk

```

If there is no configured ControlSNI, we use "example.org" as default control.

```Go
	if m.config.ControlSNI == "" {
		m.config.ControlSNI = "example.org"
	}

```

If there is no configured TestHelperAddress, we use [ControlSNI]:443 as the testhelper.

```Go
	if m.config.TestHelperAddress == "" {
		m.config.TestHelperAddress = net.JoinHostPort(
			m.config.ControlSNI, "443",
		)
	}

```

### DNS measurement

The first step is to do a DNS lookup for the testhelper's IP address(es).
This is the first time we are using dslx, yay!

We start off by extracting the host name from the testhelper address.

```Go
	thaddrHost, _, err := net.SplitHostPort(m.config.TestHelperAddress)
	if err != nil {
		return err
	}

```

Next, we describe the DNS measurement input. It consists of the domain to resolve,
and of additional parameters such as the idGenerator, the used logger, and the
experiment's start time.

```Go
	dnsInput := dslx.NewDomainToResolve(
		dslx.DomainName(thaddrHost),
	)

```

Next, we create a minimal runtime. This data structure helps us to manage
open connections and close them when `rt.Close` is invoked.

```Go
	rt := dslx.NewMinimalRuntime(args.Session.Logger(), args.Measurement.MeasurementStartTimeSaved)
	defer rt.Close()

```

We construct the resolver dslx function which can be - like in this case - the
system resolver, or a custom UDP resolver.

```Go
	lookupFn := dslx.DNSLookupGetaddrinfo(rt)

```

Then we apply the `dnsInput` argument to `lookupFn` to get a `dnsResult`.

```Go
	dnsResult := lookupFn.Apply(ctx, dslx.NewMaybeWithValue(dnsInput))

```

If there was an error during the DNS step, we cannot continue with the
experiment, so we set the failure and return.

Note that we return `nil` because this error is an experimental error
as opposed to a fundamental error (e.g., not being able to parse the
test helper address). When we return nil from this function, the OONI
core will submit the measurement to the backend (unless this functionality
has been disabled by the user). Conversely, returning an error causes the
OONI core to mark the experimental as fundamentally failed and will not
submit the measurement to the OONI backend.

```Go
	if dnsResult.Error != nil {
		tk.Result = "anomaly.test_helper_unreachable"
		return nil
	}

```

We convert the result of the DNS lookup to a unique set of IP addresses. The
output of getaddrinfo should already be a set of unique addresses, but we could
have used dslx to run several resolvers in parallel, in which case this
functionality would have been handy. This is the reason why the dslx interface
to produce addresses from DNS results always ensures they are unique.

```Go
	addresses := dslx.NewAddressSet(dnsResult)
```

### Endpoint measurements: Target and Control

We now want to perform two endpoint measurements, for target and control SNI respectively.
For each endpoint measurement we want to do two steps, i.e.
connect to the testhelper via TCP (three-way-handshake), and
establish a TLS connection to the testhelper (TLS handshake) with the target/control SNI.
As we learned in the introduction, these steps can be implemented using the building
blocks provided by dslx.

Similar to the DNS step, we first describe the input for the endpoint measurement.
The input contains the used network (we want to have "tcp" here),
the endpoint domain, the idGenerator, logger, and measurement start time.
For this simple experiment version, we will only consider the first endpoint.

(As a reminder, in OONI we use "endpoint" to refer to the combination of
the protocol, address, and port three-tuple.)

```Go
	endpoints := addresses.ToEndpoints(
		dslx.EndpointNetwork("tcp"),
		dslx.EndpointPort(443),
		dslx.EndpointOptionDomain(m.config.TestHelperAddress),
	)
	runtimex.Assert(len(endpoints) >= 1, "expected at least one endpoint here")
	endpoint := endpoints[0]

```

In the following we compose step-by-step measurement "pipelines",
represented by `dslx` functions.

For the target SNI measurement, we create a composed dslx function
that contains two building blocks: TCP connect and TLS Handshake.
For the TLS Handshake we use `TLSHandshakeOptionServerName` to specify the
target SNI to be used within the TLS Client Hello.

```Go
	pipelineTarget := dslx.Compose2(
		dslx.TCPConnect(rt),
		dslx.TLSHandshake(
			rt,
			dslx.TLSHandshakeOptionServerName(targetSNI),
		),
	)

```

For the control SNI measurement, the pipeline looks the same, except we
specify the *control* SNI to be used within the TLS Client Hello.

```Go
	pipelineControl := dslx.Compose2(
		dslx.TCPConnect(rt),
		dslx.TLSHandshake(
			rt,
			dslx.TLSHandshakeOptionServerName(m.config.ControlSNI),
		),
	)

```

We run the endpoint measurements by applying both measurement pipelines, using the
endpoint as input. The result of a single endpoint measurement is stored in a
data structure called `Maybe`, which contains either the endpoint measurement result
(on success) or an error (in case of failure).

```Go
	var targetResult *dslx.Maybe[*dslx.TLSConnection] = pipelineTarget.Apply(ctx, dslx.NewMaybeWithValue(endpoint))
	var controlResult *dslx.Maybe[*dslx.TLSConnection] = pipelineControl.Apply(ctx, dslx.NewMaybeWithValue(endpoint))

```

### Classify and store the measurement results

The `Target` and `Control` fields in the experiment's `TestKeys` contain the
measurement results. Let's go ahead and create the subresults and fill them with
the respective SNI and the testhelper address.

```Go
	tk.Target = Subresult{
		SNI:       targetSNI,
		THAddress: m.config.TestHelperAddress,
	}
	tk.Control = Subresult{
		SNI:       m.config.ControlSNI,
		THAddress: m.config.TestHelperAddress,
	}

```

We assume everything went well and set `TestKeys.Result` to the success value. We will
change our determination later after inspecting the results.

```Go
	tk.Result = "success.got_server_hello"

```

We inspect the target error to classify `TestKeys.Result`, and store the target
failure, if any. We actually expect to see an "ssl_invalid_hostname" error for the
target measurement because the TLS server (e.g., "www.example.com") *should not*
have a valid certificate for the target (e.g., "dns.google"). This error, therefore,
does *not* mean that there is censorship against the target SNI in the network
since seeing this error means that we received the certificate from the client and
completed the handshake, only to find we received an invalid certificate.

If there is a timeout error and the control measurement also failed, we assume
the testhelper is (temporarily) unreachable. (Note that this classification
of `TestKeys.Result` is simplified and differs from the experiment's specification.)

```Go
	if targetResult.Error != nil {
		failure := targetResult.Error.Error()
		tk.Target.Failure = &failure
		if failure != "ssl_invalid_hostname" {
			tk.Result = "interference"
		}
		if failure == "generic_timeout_error" && controlResult.Error != nil {
			tk.Result = "anomaly.test_helper_unreachable"
		}
	}

```

Store the control failure if any.

```Go
	if controlResult.Error != nil {
		failure := controlResult.Error.Error()
		tk.Control.Failure = &failure
	}

```

### Return

Finally, we can return, as the measurement ran successfully.

```Go
	return nil
}

```

## Run the experiment

This tutorial chapter is part of the set of experiments you can run from
the command line using the `miniooni` command. We registered this experiment
as "simple_sni" in the `internal/registry` package.

To test the code and run a measurement run:

```shell
go run ./internal/cmd/miniooni -i "ooni.org" -o "measurement.json" -n simple_sni
```

Checkout the output in "measurement.json". You should see how different
pipelines get assigned distinct `transaction_id` values.


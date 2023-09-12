
# Chapter IV: writing minimal torsf experiment

In this chapter we will replace the code written in the previous
chapter that simulates running the torsf experiment with code that
uses the `ooni/probe-cli` library to run the real experiment.

(This file is auto-generated from the corresponding source file,
so make sure you don't edit it manually.)

## Updating the imports

We need to update the imports of `torsf.go` first to look like this:

```Go

import (
```

These are standard library imports.

```Go
	"context"
	"path"
	"time"

```

As we have already seen, the `model` package defines the
generic data model used by all experiments.

```Go
	"github.com/ooni/probe-cli/v3/internal/model"

```

The `tracex` package contains code used to format internal
measurements representations to the OONI data format.

```Go
	"github.com/ooni/probe-cli/v3/internal/legacy/tracex"

```

The `ptx` package contains pluggable transport code. It includes
code to dial with obfs4 and snowflake and code to create a
pluggable transport listener.

```Go
	"github.com/ooni/probe-cli/v3/internal/ptx"

```

The `tunnel` package contains code to create a tunnel. We will
use this package to start a `tor` tunnel, which executes the `tor`
binary using specified command line arguments.

```Go
	"github.com/ooni/probe-cli/v3/internal/tunnel"
)

```


## Rewriting the run method

Let us now rewrite the `run` method to run a real `torsf`
test rather than just pretending to do it.

```Go
func (m *Measurer) run(ctx context.Context,
	sess model.ExperimentSession, testkeys *TestKeys, errch chan<- error) {
```

As a first step, we create a dialer for snowflake using the
`ptx` package. This dialer will allow us to create a `net.Conn`-like
network connection where traffic is sent using the Snowflake
pluggable transport. There are several optional fields in
`SnowflakeDialer`; the `NewSnowflakeDialer` constructor will
give us a suitable configured dialer with default settings.

```Go
	sfdialer := ptx.NewSnowflakeDialer()
```

Let us now create a listener. The `ptx.Listener` is a listener
that listens on a local port and speaks the SOCKS5 protocol. When
tor connect to this port, the listener will forward the traffic
to the Snowflake dialer we previously created. We will also
use the session's logger to emit logging messages.

```Go
	ptl := &ptx.Listener{
		PTDialer: sfdialer,
		Logger:   sess.Logger(),
	}
```

Now we start the listener. This entails opening a port on the
local host. If this operation fails, we return an error. In fact,
a failure here means a hard error that prevented us from even
starting the experiment. Therefore, it's consistent with the
`Run`'s expectations to return an error here.

```Go
	if err := ptl.Start(); err != nil {
		testkeys.Failure = tracex.NewFailure(err)
		errch <- err
		return
	}
	defer ptl.Stop()
```

Next, we start `tor` using the `tunnel` package. Note how we
pass specific `TorArgs` that cause `tor` to know about the
pluggable transport created by `ptl` and `sfdialer`.

```Go
	tun, _, err := tunnel.Start(ctx, &tunnel.Config{
		Name:      "tor",
		Session:   sess,
		TunnelDir: path.Join(sess.TempDir(), "torsf"),
		Logger:    sess.Logger(),
		TorArgs: []string{
			"UseBridges", "1",
			"ClientTransportPlugin", ptl.AsClientTransportPluginArgument(),
			"Bridge", sfdialer.AsBridgeArgument(),
		},
	})
```

In case of error, we convert `err` to a OONI failure using
the `NewFailure` function of `tracex`. This function reduces
Go error strings to the error strings used by OONI. You can
read the [errors spec](https://github.com/ooni/spec/blob/master/data-formats/df-007-errors.md)
at the [github.com/ooni/spec repo](https://github.com/ooni/spec).

Note that, in this case, we return `nil` to the caller, because
a failure here is not a fundamental failure in running the
experiment, but rather a possibly interesting anomaly.

```Go
	if err != nil {
		testkeys.Failure = tracex.NewFailure(err)
		errch <- nil
		return
	}
```

Otherwise, we successfully created a tor tunnel using Snowflake,
so we just close the tunnel and record the bootstrap time.

```Go
	defer tun.Stop()
	testkeys.BootstrapTime = tun.BootstrapTime().Seconds()
	errch <- nil
}

```

## Running the code

We can now run the code as follows to obtain:

```
$ go run ./experiment/torsf/chapter04 | tail -n 1 | jq
[...]
Jun 21 23:40:50.000 [notice] Bootstrapped 100% (done): Done
2021/06/21 23:40:50  info [100.0%] torsf experiment is finished
Jun 21 23:40:50.000 [notice] Catching signal TERM, exiting cleanly
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
    "bootstrap_time": 48.122813,
    "failure": null
  },
  "test_name": "",
  "test_runtime": 0,
  "test_start_time": "",
  "test_version": ""
}
```

## Concluding remarks

Congratulations, we have now rewritten together (a simplified version of)
the `torsf` experiment! In this journey, we have learned how experiments
interact with the rest of OONI Probe, how they are typically organized,
and how to use lower-level libraries to implement them.


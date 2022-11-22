
# Chapter III: starting to simulate a real torsf experiment

In this chapter we will improve upon what we did in the previous
chapter by creating runner code for the `torsf` experiment. We will
not, yet, run the real experiment, but we will instead write
simple code that pretends to run a `tor` bootstrap using snowflake.

(This file is auto-generated from the corresponding source file,
so make sure you don't edit it manually.)

### The TestKeys structure

Let us start by defining the `TestKeys` structure that contains
the experiment specific results. As we have already seen in
Chapter I, this structure must contain two fields. The bootstrap
time for the experiment and the failure.

```Go
type TestKeys struct {
	BootstrapTime float64 `json:"bootstrap_time"`
	Failure       *string `json:"failure"`
}

```

### Rewriting the Run method

Next we will rewrite the Run method. We will arrange for this
method to fill the `measurement`, to setup the timeout, and to
print periodic updates via the `callbacks`. We will defer the
real work to a private function called `run`.

```Go
func (m *Measurer) Run(ctx context.Context, args *model.ExperimentArgs) error {
	callbacks := args.Callbacks
	measurement := args.Measurement
	sess := args.Session
```

Let's create an instance of `TestKeys` and let's modify
the `measurement` to refer to such an instance.

```Go
	testkeys := &TestKeys{}
	measurement.TestKeys = testkeys
```

Next, we record the current time and we modify the
context to have a timeout after 300 seconds. Because
Snowflake *may* take a long time to bootstrap, we
need to specify a generous timeout here.

```Go
	start := time.Now()
	const maxRuntime = 300 * time.Second
	ctx, cancel := context.WithTimeout(ctx, maxRuntime)
	defer cancel()
```

Okay, now we are ready to defer the real work to
the internal `run` function. We first create a
channel to receive the result of `run`. Then, we
create a ticker to emit periodic updates. We
emit an update every 250 milliseconds, which is
a reasonably smooth way of increasing a progress
bar (progress is indeed used to move progress bars
both in OONI Probe Desktop and mobile.)

```Go
	errch := make(chan error)
	ticker := time.NewTicker(250 * time.Millisecond)
	defer ticker.Stop()
```

Now we defer the real work to `run`, which will
run in a background goroutine.

```Go
	go m.run(ctx, sess, testkeys, errch)
```

While `run` is running, we loop and check which
channel has become ready.

If the `errch` channel is ready, it means that `run` is
terminated, so we return to the caller.

Instead, if `ticker.C` is ready, we emit a progress
update using the `callbacks`.

```Go
	for {
		select {
		case err := <-errch:
			callbacks.OnProgress(1.0, "torsf experiment is finished")
			return err
		case <-ticker.C:
			progress := time.Since(start).Seconds() / maxRuntime.Seconds()
			callbacks.OnProgress(progress, "torsf experiment is running")
		}
	}
}

```

### The run function

We will now implement the `run` function. For now, this function
will not do any real work, but it will just pretend to do work.

Note how we sleep for some time, set the `BootstrapTime` field
of the `TestKeys`, and then return using `errch`.

```Go
func (m *Measurer) run(ctx context.Context,
	sess model.ExperimentSession, testkeys *TestKeys, errch chan<- error) {
	fakeBootstrapTime := 10 * time.Second
	time.Sleep(fakeBootstrapTime)
	testkeys.BootstrapTime = fakeBootstrapTime.Seconds()
	errch <- nil
}

```

## Running the code

It's now time to run the new code we've written:

```
$ go run ./experiment/torsf/chapter03 | jq
2021/06/21 21:21:18  info [  0.1%] torsf experiment is running
2021/06/21 21:21:19  info [  0.2%] torsf experiment is running
[...]
2021/06/21 21:21:28  info [  3.3%] torsf experiment is running
2021/06/21 21:21:28  info [100.0%] torsf experiment is finished
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
  "bootstrap_time": 10,
  "failure": null
  },
  "test_name": "",
  "test_runtime": 0,
  "test_start_time": "",
  "test_version": ""
}
```

You see that now we're filling the bootstrap time and we're
also printing progress using `callbacks`.

In the next chapter, we'll replace the stub `run` implementation
with a real implementation using Snowflake.


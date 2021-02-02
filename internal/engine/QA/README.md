# Quality Assurance scripts

This directory contains quality assurance scripts that use Jafar to
ensure that OONI implementations behave. These scripts take on the
command line as argument the path to a binary with a OONI Probe v2.x
like command line interface. We do not care about full compatibility
but rather about having enough similar flags that running these tools
in parallel is not too much of a burden for us.

Tools with this shallow-compatible CLI are:

1. `github.com/ooni/probe-legacy`
2. `github.com/measurement-kit/measurement-kit/src/measurement_kit`
3. `github.com/ooni/probe-engine/cmd/miniooni`

## Run QA on a Linux system

These scripts assume you're on a Linux system with `iptables`, `bash`,
`python3`, and possibly a bunch of other tools installed.

To start the QA script, run this command:

```bash
sudo ./QA/$nettest.py $ooni_exe
```

where `$nettest` is the nettest name (e.g. `telegram`) and `$ooni_exe`
is the OONI Probe v2.x compatible binary to test.

The Python script needs to run as root. Note however that sudo will also
be used to run `$ooni_exe` with the privileges of the `nobody` user.

## Run QA using a docker container

Run test in a suitable Docker container using:

```bash
./QA/rundocker.sh $nettest
```

Note that this will run a `--privileged` docker container. This will
eventually run the Python script you would run on Linux.

For now, the docker scripts only perform QA of `miniooni`.

## Diagnosing issues

The Python script that performs the QA runs a specific OONI test under
different failure conditions and stops at the first unexpected value found
in the resulting JSONL report. You can infer what went wrong by reading
the output of the `$ooni_exe` command itself, which should be above the point
where the Python script stopped, as well as by inspecting the JSONL file on
disk. By convention such file is named `$nettest.jsonl` and only contains
the result of the last run of `$nettest`.

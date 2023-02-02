# Quality assurance scripts

This directory contains quality assurance scripts that use Jafar to
ensure that OONI implementations behave. These scripts work with miniooni.

## Run QA using a docker container

Run test in a suitable Docker container using:

```bash
./QA/rundocker.bash $nettest
```

Note that this will run a `--privileged` docker container.

## Diagnosing issues

The Python script that performs the QA runs a specific OONI test under
different failure conditions and stops at the first unexpected value found
in the resulting JSONL report. You can infer what went wrong by reading
the output of the `miniooni` command itself, which should be above the point
where the Python script stopped, as well as by inspecting the JSONL file on
disk. By convention such file is named `$nettest.jsonl` and only contains
the result of the last run of `$nettest`.

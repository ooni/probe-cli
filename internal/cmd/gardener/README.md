# Gardener

The gardener is a tool that helps curating the [test-lists](
https://github.com/citizenlab/test-lists).

## Build instructions

```bash
go build -v ./internal/cmd/gardener
```

## Usage

### Cloning the test-lists repository

```bash
./garderner sync
```

This command clones the most recent version of the [test-lists](
https://github.com/citizenlab/test-lists) repository in a local
directory. If the repository has been already cloned previously, this
command will delete the local directory and fetch it again.

### Generating a DNS data quality report

```bash
./gardener dnsreport
```

This command generates a `dnsreport.sqlite3` database containing
an entry for each URL of the test list. We will record the original
file name, the file line, the URL, and the result of the DNS
lookup for the URL's domain. If the DNS lookup failed, we also
include the counters returned by the aggregation API for the given
input URL, which provides information useful to determine whether
we actually want to drop this URL from the test list.

The resolver used by this command is the same used by the Web
Connectivity test helper, so its results _should_ be consistent
with the ones observed by the test helper itself.

You can interrupt this command at any time. Re-running it will
only measure the unmeasured URLs as long as you keep the
`dnsreport.sqlite3` file around.

When done, this command produces a `dnsreport.csv` file containing
summary information about the expired domains.

### Remove the most obvious expired domains

```bash
./gardener dnsfix
```

This command uses the `dnsreport.csv` file and applies _simple_ rules
to only remove the most-safe-to-remove URLs from the test lists.

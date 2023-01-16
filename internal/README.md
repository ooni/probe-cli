# OONI Probe Measurement Engine

This directory contains private Go packages. Collectively these
package are the measurement engine that empowers OONI Probe. Most
of the code in this package derives from the [ooni/probe-engine](
https://github.com/ooni/probe-engine) discontinued repository.

## Reading documentation from the command line

You can read the Go documentation of a package by using `go doc -all`.

For example:

```bash
go doc -all ./internal/netxlite
```

## Reading documentation in the browser

You can install the `pkgsite` tool using this command:

```bash
go install golang.org/x/pkgsite@latest
```

To run `pkgsite`, use:

```bash
pkgsite
```

Then visit http://127.0.0.1:8080/github.com/ooni/probe-cli/v3 with
your browser to browse the documentation.

## Getting information about a package

Use the `go list -json` subcommand. For example:

```
go list -json ./internal/netxlite
```

## Generating a dependency graph

You can get a graph of the dependencies using [kisielk/godepgraph](https://github.com/kisielk/godepgraph).

For example:

```bash
godepgraph -s -novendor -p golang.org,gitlab.com ./internal/engine | dot -Tpng -o deps.png
```

You can further tweak which packages to exclude by appending
prefixes to the list passed to the `-p` flag.

## Tutorials

The [tutorial](tutorial) package contains tutorials on writing new experiments,
using measurements libraries, and networking code.

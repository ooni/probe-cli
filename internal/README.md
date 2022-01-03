# Directory github.com/ooni/probe-cli/internal

This directory contains private Go packages.

As a reminder, you can always check the Go documentation of
a package by using

```bash
go doc -all ./internal/$package
```

where `$package` is the name of the package.

Some notable packages:

- [model](model) contains the interfaces and data model shared
by most packages inside this directory;

- [netxlite](netxlite) is the underlying networking library;

- [tutorial](tutorial) contains tutorials on writing new experiments,
using measurements libraries, and networking code.

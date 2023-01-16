[![OONI Probe Android](docs/logo.png)](https://ooni.org)

<div align="center">

[![GoDoc](https://godoc.org/github.com/ooni/probe-cli?status.svg)](https://godoc.org/github.com/ooni/probe-cli) [![Coverage Status](https://coveralls.io/repos/github/ooni/probe-cli/badge.svg?branch=master)](https://coveralls.io/github/ooni/probe-cli?branch=master) [![Slack](https://slack.openobservatory.org/badge.svg)](https://slack.openobservatory.org/)

## OONI Probe Measurement Engine and CLI

**[Install instructions »](https://ooni.org/install/cli)**

[User guide](https://ooni.org/support/ooni-probe-cli) · [API docs](https://godoc.org/github.com/ooni/probe-cli) · [Report bug](https://github.com/ooni/probe/issues/new?labels=ooni/probe-cli&assignee=bassosimone) · [Request feature](https://github.com/ooni/probe/issues/new?labels=ooni/probe-cli&assignee=bassosimone)

</div>

<details>
  <summary>Table of Contents</summary>
  <ol>
    <li><a href="#about-this-project">About this project</a></li>
    <li><a href="#install-instructions">Install instructions</a></li>
    <li><a href="#build-instructions>Build instructions<a></li>
    <li><a href="#contributing">Contributing</a></li>
    <li><a href="#license">License</a></li>
    <li><a href="#updating-dependencies">Updating dependencies</a></li>
    <li><a href="#releasing">Releasing<a></li>
    <li><a href="#semantic-versioning-policy">Semantic versioning policy<a></li>
  </ol>
</details>

<hr>

## About this project

The [Open Observatory of Network Interference](https://ooni.org) (OONI)
is a non-profit free software project that aims to empower decentralized
efforts in documenting Internet censorship around the world.

This repository contains the following Go packages:

1. the command line client ([cmd/ooniprobe](cmd/ooniprobe));

2. the test helper server ([internal/cmd/oohelperd](internal/cmd/oohelperd));

3. the mobile library ([pkg/oonimkall](pkg/oonimkall));

4. the measurement-engine library ([internal](internal)).

Every top-level directory in this repository contains an explanatory README file.

## Install instructions

Follow the instructions at [ooni.org/install/cli](https://ooni.org/install/cli)
to install `ooniprobe` precompiled binaries for Windows, macOS, and
Debian/Ubuntu. Once `ooniprobe` is installed, refer to the
[user guide](https://ooni.org/support/ooni-probe-cli). Run `ooniprobe --help`
to get brief help instructions.

## Build instructions

To build `ooniprobe` or other binaries manually, you need to
install and use the Go version mentioned by the
[GOVERSION](GOVERSION) file. Using a different version of
Go _may_ work as intended but is not recommended: we depend
on packages forked from the standard library; so, it is
more robust to use the same version of Go from which
we forked those packages from.

You will also need a C compiler. On Linux and other Unix systems
both GCC and Clang will work. If you're using Windows, we
recommend installing Ubuntu or Debian on [the Windows Subsystem
for Linux](https://learn.microsoft.com/en-us/windows/wsl/install).
If you're targeting Windows, you should install the
mingw-w64 cross-compiler.

Once you have installed the correct Go version and a C compiler,
you can compile `ooniprobe` using:

```bash
go build -v -ldflags '-s -w' ./cmd/ooniprobe
```

This command will generate a stripped binary called `ooniprobe`
in the toplevel directory.

Likewise, you can compile `miniooni` using:

```bash
go build -v -ldflags '-s -w' ./internal/cmd/miniooni
```

This command will generate a stripped binary called `miniooni`
in the toplevel directory.

And `oohelperd` using:

```bash
go build -v -ldflags '-s -w' ./internal/cmd/oohelperd
```

This command will generate a stripped binary called `oohelperd`
in the toplevel directory.

## Contributing

Please, see [CONTRIBUTING.md](CONTRIBUTING.md).

## License

```
SPDX-License-Identifier: GPL-3.0-or-later
```

## Updating dependencies

```bash
go get -t -u -v ./... && go mod tidy
```

## Releasing

Create an issue according to [the routine release template](
https://github.com/ooni/probe/blob/master/.github/ISSUE_TEMPLATE/routine-sprint-releases.md)
and perform any item inside the check-list.

We build releases using [Makefile](Makefile), which requires GNU make. Run
`make help` for detailed usage.

## Semantic versioning policy

The mobile library is a public package for technical reasons. Go mobile tools require
a public package to build from. Yet, we don't consider API breakages happening in
such a package to be sufficient to bump our major version number. For us, the mobile
library is just a mean to implement OONI Probe Android and OONI Probe iOS. We'll
only bump the major version number if we change `./cmd/ooniprobe`'s CLI.

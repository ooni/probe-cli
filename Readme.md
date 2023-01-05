# OONI Probe Client Library and CLI

[![GoDoc](https://godoc.org/github.com/ooni/probe-cli?status.svg)](https://godoc.org/github.com/ooni/probe-cli) [![Coverage Status](https://coveralls.io/repos/github/ooni/probe-cli/badge.svg?branch=master)](https://coveralls.io/github/ooni/probe-cli?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/ooni/probe-cli/v3)](https://goreportcard.com/report/github.com/ooni/probe-cli/v3)

The [Open Observatory of Network Interference](https://ooni.org) (OONI) is a non-profit free software project
that aims to empower decentralized efforts in documenting
Internet censorship around the world.

This repository contains core OONI tools written in Go:

- the CLI client ([cmd/ooniprobe](cmd/ooniprobe));

- the test helper server ([internal/cmd/oohelperd](internal/cmd/oohelperd));

- the mobile library ([pkg/oonimkall](pkg/oonimkall));

- the OONI Probe engine (inside [internal](internal)).

Every top-level directory in this repository contains an explanatory README file. You
may also notice that some internal packages live under [internal/engine](internal/engine)
while most others are top-level. This is part of [a long-standing refactoring](
https://github.com/ooni/probe/issues/2115) started when we merged
https://github.com/ooni/probe-engine into this repository. We'll slowly
ensure that all packages inside `engine` are moved out of it and inside `internal`.

## Semantic versioning policy

The mobile library is a public package for technical reasons. Go mobile tools require
a public package to build from. Yet, we don't consider API breakages happening in
such a package to be sufficient to bump our major version number. For us, the mobile
library is just a mean to implement OONI Probe Android and OONI Probe iOS. We'll
only bump the major version number if we change `./cmd/ooniprobe`'s CLI.

## License

```
SPDX-License-Identifier: GPL-3.0-or-later
```

## User setup

Please, follow the instructions at [ooni.org/install/cli](https://ooni.org/install/cli)
to install `ooniprobe`. If we do not support your use case, please let us know. Once
`ooniprobe` is installed, try `ooniprobe help` to get interactive help.

## Reporting issues

Report issues at [github.com/ooni/probe](
https://github.com/ooni/probe/issues/new?labels=ooni/probe-cli&assignee=bassosimone).
Please, make sure you add the `ooni/probe-cli` label.

## Build instructions

Be sure you have:

1. the golang version mentioned inside the [GOVERSION](GOVERSION) file;

2. a C compiler (Mingw-w64 for Windows).

### ooniprobe

Ooniprobe is the official CLI client. Compile using:

```bash
go build -v -ldflags "-s -w" ./cmd/ooniprobe
```

This will generate a binary called `ooniprobe` in the current directory.

### miniooni

Miniooni is the experimental OONI client used for research. Compile using:

```bash
go build -v -ldflags "-s -w" ./internal/cmd/miniooni
```

This will generate a binary called `miniooni` in the current directory.

### oohelperd

Oohelperd is the test helper server. Compile using:

```bash
go build -v -ldflags "-s -w" ./internal/cmd/oohelperd
```

This will generate a binary called `oohelperd` in the current directory.

## Specifications

Every nettest (aka experiment) implemented in this repository has a companion
spec in [ooni/spec](https://github.com/ooni/spec).

## Contributing

Please, see [CONTRIBUTING.md](CONTRIBUTING.md).

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

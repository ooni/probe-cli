# OONI Probe Client Library and CLI

[![GoDoc](https://godoc.org/github.com/ooni/probe-cli?status.svg)](https://godoc.org/github.com/ooni/probe-cli) [![Short Tests Status](https://github.com/ooni/probe-cli/workflows/shorttests/badge.svg)](https://github.com/ooni/probe-cli/actions?query=workflow%3Ashorttests) [![Coverage Status](https://coveralls.io/repos/github/ooni/probe-cli/badge.svg?branch=master)](https://coveralls.io/github/ooni/probe-cli?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/ooni/probe-cli)](https://goreportcard.com/report/github.com/ooni/probe-cli) [![linux-debian-packages](https://github.com/ooni/probe-cli/workflows/linux-debian-packages/badge.svg)](https://github.com/ooni/probe-cli/actions?query=workflow%3Alinux-debian-packages)

The [Open Observatory of Network Interference](https://ooni.org) (OONI) is a free software project
that aims to empower decentralized efforts in increasing transparency of
Internet censorship around the world.

This repository contains core OONI tools written in Go:

- the CLI client ([cmd/ooniprobe](cmd/ooniprobe));

- the mobile library ([pkg/oonimkall](pkg/oonimkall));

- the test helper server ([internal/cmd/oohelperd](internal/cmd/oohelperd));

- and all the related support packages (inside [internal](internal)).

Every top-level directory in this repository contains an explanatory README file.

## User setup

Please, follow the instructions at [ooni.org/install/cli](https://ooni.org/install/cli)
to install `ooniprobe`. If we do not support your use case, please let us know. Once
`ooniprobe` is installed, try `ooniprobe help` to get interactive help.

## Reporting issues

Report issues at [github.com/ooni/probe](
https://github.com/ooni/probe/issues/new?labels=ooni/probe-cli&assignee=bassosimone).
Please, make sure you add the `ooni/probe-cli` label.

## Build instructions

### ooniprobe

Be sure you have golang >= 1.17 and a C compiler (Mingw-w64 for Windows). You
can build using:

```bash
go build -v ./cmd/ooniprobe
```

This will generate a binary called `ooniprobe` in the current directory.

### Android bindings

Make sure you have GNU make installed, then run:

```bash
./mk android
```

to build bindings for Android. (Add `OONI_PSIPHON_TAGS=""` if you
cannot clone private repositories in the https://github.com/ooni namespace.)

The generated bindings are (manually) pushed to the Maven Central package
repository. The instructions explaining how to integrate these bindings
are published along with the release notes.

### iOS bindings

Make sure you have GNU make installed, then run:

```bash
./mk ios
```

to build bindings for iOS. (Add `OONI_PSIPHON_TAGS=""` if you
cannot clone private repositories in the https://github.com/ooni namespace.)

The generated bindings are (manually) added to GitHub releases. The instructions
explaining how to integrate these bindings are published along with the release notes.

### miniooni

Miniooni is the experimental OONI client used for research. Compile using:

```bash
go build -v ./internal/cmd/miniooni
```

This will generate a binary called `miniooni` in the current directory.

### oohelperd

Oohelperd is the test helper server. Compile using:

```bash
go build -v ./internal/cmd/oohelperd
```

This will generate a binary called `oohelperd` in the current directory.

## Specifications

Every nettest (aka experiment) implemented in this repository has a companion
spec in the [ooni/spec](https://github.com/ooni/spec) repository.

## Contributing

Please, see [CONTRIBUTING.md](CONTRIBUTING.md).

## Updating dependencies

```bash
go get -u -v ./... && go mod tidy
```

## Releasing

Create an issue according to [the routine release template](
https://github.com/ooni/probe/blob/master/.github/ISSUE_TEMPLATE/routine-sprint-releases.md)
and perform any item inside the check-list.

We build releases using `./mk`, which requires GNU make. Try
the `./mk help|less` command for detailed usage.

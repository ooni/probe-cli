# OONI Probe Client Library and CLI

* Documentation: [![GoDoc](https://godoc.org/github.com/ooni/probe-cli?status.svg)](https://godoc.org/github.com/ooni/probe-cli)

* `go test -race -short ./...` status: [![Short Tests Status](https://github.com/ooni/probe-cli/workflows/shorttests/badge.svg)](https://github.com/ooni/probe-cli/actions?query=workflow%3Ashorttests)

* `go test -race ./...` status: [![All Tests Status](https://github.com/ooni/probe-cli/workflows/alltests/badge.svg)](https://github.com/ooni/probe-cli/actions?query=workflow%3Aalltests)

* Code coverage for `-short` tests: [![Coverage Status](https://coveralls.io/repos/github/ooni/probe-cli/badge.svg?branch=master)](https://coveralls.io/github/ooni/probe-cli?branch=master)

* Go Report Card: [![Go Report Card](https://goreportcard.com/badge/github.com/ooni/probe-cli)](https://goreportcard.com/report/github.com/ooni/probe-cli)

* Debian package builds: [![linux-debian-packages](https://github.com/ooni/probe-cli/workflows/linux-debian-packages/badge.svg)](https://github.com/ooni/probe-cli/actions?query=workflow%3Alinux-debian-packages)

* Open issues: [![GitHub issues by-label](https://img.shields.io/github/issues/ooni/probe/ooni/probe-cli?style=plastic)](https://github.com/ooni/probe/labels/ooni%2Fprobe-cli)

The next generation OONI Probe: client library and Command Line Interface.

## User setup

Please, follow the instructions at [ooni.org/install/cli](https://ooni.org/install/cli)
to install `ooniprobe`. If we do not support your use case, please let us know. Once
`ooniprobe` is installed, try `ooniprobe help` to get interactive help.

## Reporting issues

Report issues at [github.com/ooni/probe](
https://github.com/ooni/probe/issues/new?labels=ooni/probe-cli&assignee=bassosimone).
Please, make sure you add the `ooni/probe-cli` label.

## Repository organization

Every top-level directory contains an explanatory README file.

## ooniprobe

Be sure you have golang >= 1.16 and a C compiler (Mingw-w64 for Windows). You
can build using:

```bash
go build -v ./cmd/ooniprobe
```

This will generate a binary called `ooniprobe` in the current directory.

## Android bindings

Make sure you have GNU make installed, then run:

```bash
./mk android
```

to build bindings for Android. (Add `OONI_PSIPHON_TAGS=""` if you
cannot clone private repositories in the https://github.com/ooni namespace.)

The generated bindings are (manually) pushed to the Maven Central package
repository. The instructions explaining how to integrate these bindings
are published along with the release notes.

## iOS bindings

Make sure you have GNU make installed, then run:

```bash
./mk ios
```

to build bindings for iOS. (Add `OONI_PSIPHON_TAGS=""` if you
cannot clone private repositories in the https://github.com/ooni namespace.)

The generated bindings are (manually) added to GitHub releases. The instructions
explaining how to integrate these bindings are published along with the release notes.

## miniooni

Miniooni is the experimental OONI client used for research. Compile using:

```bash
go build -v ./internal/cmd/miniooni
```

This will generate a binary called `miniooni` in the current directory.

## Specification

Every nettest (aka experiment) implemented in this repository has a companion
spec in the [ooni/spec](https://github.com/ooni/spec) repository.


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

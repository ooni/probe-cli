# OONI Probe Client Library and CLI

[![GoDoc](https://godoc.org/github.com/ooni/probe-cli?status.svg)](https://godoc.org/github.com/ooni/probe-cli) [![Short Tests Status](https://github.com/ooni/probe-cli/workflows/shorttests/badge.svg)](https://github.com/ooni/probe-cli/actions?query=workflow%3Ashorttests) [![All Tests Status](https://github.com/ooni/probe-cli/workflows/alltests/badge.svg)](https://github.com/ooni/probe-cli/actions?query=workflow%3Aalltests) [![Coverage Status](https://coveralls.io/repos/github/ooni/probe-cli/badge.svg?branch=master)](https://coveralls.io/github/ooni/probe-cli?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/ooni/probe-cli)](https://goreportcard.com/report/github.com/ooni/probe-cli) [![linux-debian-packages](https://github.com/ooni/probe-cli/workflows/linux-debian-packages/badge.svg)](https://github.com/ooni/probe-cli/actions?query=workflow%3Alinux-debian-packages) [![GitHub issues by-label](https://img.shields.io/github/issues/ooni/probe/ooni/probe-cli?style=plastic)](https://github.com/ooni/probe/labels/ooni%2Fprobe-cli)

The next generation OONI Probe: client library and Command Line Interface.

## User setup

Please, follow the instructions at [ooni.org/install/cli](https://ooni.org/install/cli)
to install `ooniprobe`. If we do not support your use case, please let us know.

Once `ooniprobe` is installed, try `ooniprobe help` to get interactive help.

## Reporting issues

Please, report issues with this codebase at [github.com/ooni/probe](
https://github.com/ooni/probe/issues/new?labels=ooni/probe-cli&assignee=bassosimone).
Please, make sure you tag such issues using the `ooni/probe-cli` label.

## Repository organization

Every top-level directory contains an explanatory README file.

## OONIProbe

Be sure you have golang >= 1.16 and a C compiler (when developing for Windows, you
need Mingw-w64 installed). You can build using:

```bash
go build -v ./cmd/ooniprobe
```

This will generate a binary called `ooniprobe` in the current directory.

## Android bindings

Make sure you have Python 3.8+ installed, then run:

```bash
./make -t android
```

Builds bindings for Android. (Add `----disable-embedding-psiphon-config` if you
cannot clone private repositories in the https://github.com/ooni namespace.)

The generated bindings are (manually) pushed to the Maven Central package
repository. The instructions explaining how to integrate these bindings
are published along with the release notes.

## iOS bindings

Make sure you have Python 3.8+ installed, then run:

```bash
./make -t ios
```

Builds bindings for iOS. (Add `----disable-embedding-psiphon-config` if you
cannot clone private repositories in the https://github.com/ooni namespace.)

The generated bindings are (manually) added to GitHub releases. The instructions
explaining how to integrate these bindings are published along with the release notes.

## miniooni

Miniooni is the experimental OONI client used for research. Compile using:

```bash
go build -v ./internal/cmd/miniooni
```

This will generate a binary called `miniooni` in the current directory.

## Updating dependencies

```bash
go get -u -v ./... && go mod tidy
```

## Releasing

Create an issue according to [the routine release template](
https://github.com/ooni/probe/blob/master/.github/ISSUE_TEMPLATE/routine-sprint-releases.md)
and perform any item inside the check-list.

We build releases using `./make`, which requires Python3.8+.

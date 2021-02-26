# OONI Probe CLI

[![GoDoc](https://godoc.org/github.com/ooni/probe-cli?status.svg)](https://godoc.org/github.com/ooni/probe-cli) [![Short Tests Status](https://github.com/ooni/probe-cli/workflows/shorttests/badge.svg)](https://github.com/ooni/probe-cli/actions?query=workflow%3Ashorttests) [![All Tests Status](https://github.com/ooni/probe-cli/workflows/alltests/badge.svg)](https://github.com/ooni/probe-cli/actions?query=workflow%3Aalltests) [![Coverage Status](https://coveralls.io/repos/github/ooni/probe-cli/badge.svg?branch=master)](https://coveralls.io/github/ooni/probe-cli?branch=master) [![Go Report Card](https://goreportcard.com/badge/github.com/ooni/probe-cli)](https://goreportcard.com/report/github.com/ooni/probe-cli) [![linux-debian-packages](https://github.com/ooni/probe-cli/workflows/linux-debian-packages/badge.svg)](https://github.com/ooni/probe-cli/actions?query=workflow%3Alinux-debian-packages) [![GitHub issues by-label](https://img.shields.io/github/issues/ooni/probe/ooni/probe-cli?style=plastic)](https://github.com/ooni/probe/labels/ooni%2Fprobe-cli)

The next generation OONI Probe Command Line Interface.

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

## Development setup

Be sure you have golang >= 1.16 and a C compiler (when developing for Windows, you
need Mingw-w64 installed).

You need to download assets first using:

```bash
go run ./internal/cmd/getresources
```

Then you can build using:

```bash
go build -v ./cmd/ooniprobe
```

This will generate a binary called `ooniprobe` in the current directory.

## Android bindings

You need to download assets first using:

```bash
go run ./internal/cmd/getresources
```

Then you can build using:

```bash
./build-android.bash
```

We automatically build Android bindings whenever commits are pushed to the
`mobile-staging` branch. Such builds could be integrated by using:

```Groovy
implementation "org.ooni:oonimkall:VERSION"
```

Where VERSION is like `2020.03.30-231914` corresponding to the
time when the build occurred.

## iOS bindings

You need to download assets first using:

```bash
go run ./internal/cmd/getresources
```

Then you can build using:

```bash
./build-ios.bash
```

We automatically build iOS bindings whenever commits are pushed to the
`mobile-staging` branch. Such builds could be integrated by using:

```ruby
pod 'oonimkall', :podspec => 'https://dl.bintray.com/ooni/ios/oonimkall-VERSION.podspec'
```

Where VERSION is like `2020.03.30-231914` corresponding to the
time when the build occurred.

## Updating dependencies

```bash
go get -u -v ./... && go mod tidy
```

## Releasing

1. update binary data as described above;

2. update `internal/version/version.go`;

3. make sure you have updated dependencies;

4. run `./build.sh release` and follow instructions.

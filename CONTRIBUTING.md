# Contributing to ooni/probe-cli

This is an open source project, and contributions are welcome! You are welcome
to open pull requests. An open pull request will be reviewed by a core
developer. The review may request you to apply changes. Once the assigned
reviewer is satisfied, they will merge the pull request.

## OONI Software Development Guidelines

Please, make sure you read [OONI Software Development Guidelines](
https://ooni.org/post/ooni-software-development-guidelines/). We try in
general to follow these guidelines when working on ooni/probe-cli. In
the unlikely case where those guidelines conflict with this document, this
document will take precedence.

## Golang Resources

We use golang as our primary language for the development of OONI Probe CLI and do check out the resources below, quite useful to read before contributing.

- [Effective Go](https://go.dev/doc/effective_go)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Concurrency](https://go.dev/blog/pipelines) and [Data races](https://go.dev/ref/mem)
- [Channels Axioms](https://dave.cheney.net/2014/03/19/channel-axioms)

## Opening issues

Please, before opening a new issue, check whether the issue or feature request
you want us to consider has not already been reported by someone else. The
issue tracker is at [github.com/ooni/probe/issues](https://github.com/ooni/probe/issues).

## PR requirements

Every pull request that introduces new functionality should feature
comprehensive test coverage. Any pull request that modifies existing
functionality should pass existing tests. What's more, any new pull
request that modifies existing functionality should not decrease the
existing code coverage.

Long-running tests should be skipped when running tests in short mode
using `go test -short`. We prefer internal testing to external
testing. We generally have a file called `foo_test.go` with tests
for every `foo.go` file. Sometimes we separate long running
integration tests in a `foo_integration_test.go` file.

If there is a top-level DESIGN.md document, make sure such document is
kept in sync with code changes you have applied.

Do not submit large PRs. A reviewer can best service your PR if the
code changes are around 200-600 lines. (It is okay to add more changes
afterwards, if the reviewer asks you to do more work; the key point
here is that the PR should be reasonably sized when the review starts.)

In this vein, we'd rather structure a complex issue as a sequence of
small PRs, than have a single large PR address it all.

As a general rule, a PR is reviewed by reading the whole diff. Let us
know if you want us to read each diff individually, if you think that's
functional to better understand your changes.

## Code style requirements

Please, use `go fmt`, `go vet`, and `golint` to check your code
contribution before submitting a pull request. Make sure your code
is documented. At the minimum document all the exported symbols.

Make sure you commit `go.mod` and `go.sum` changes. Make sure you
run `go mod tidy` to minimize such changes.

## Implementation requirements

- use `./internal/atomicx` rather than `atomic/sync`

- do not use `os/exec`, use `x/sys/execabs` or `./internal/shellx`

- use `./internal/fsx.OpenFile` when you need to open a file

- use `./internal/netxlite.ReadAllContext` instead of `io.ReadAll`
and `./internal/netxlite.CopyContext` instead of `io.Copy`

- use `./internal/model.ErrorToStringOrOK` when 
an experiment logs intermediate results

## Code testing requirements

Make sure all tests pass with `go test -race ./...` run from the
top-level directory of this repository. (Integration tests may be
flaky, so there may be some failures here and and there; we know
in particular that `./internal/cmd/jafar` is one of the usual
suspects and that it's not super pleasant to test it under Linux.)

## Writing a new OONI experiment

When you are implementing a new experiment (aka nettest), make sure
you have read the relevant spec from the [ooni/spec](
https://github.com/ooni/spec) repository. If the spec is missing,
please help the pull request reviewer to create it. If the spec is
not clear, please let us know during the review.

To get a sense of what we expect from an experiment, see the [internal/tutorial](
https://github.com/ooni/probe-cli/tree/master/internal/tutorial) tutorial

## Branching and releasing

The following diagram illustrates the overall branching and releasing
strategy loosely followed by the core team. If you are an external
contributor, you generally only care about the development part, which
is on the left-hand side of the diagram.

![branching and releasing](docs/branching.png)

Development uses the `master` branch. When we need to implement a
feature or fix a bug, we branch off of the `master` branch. We squash
and merge to include a feature or fix branch back into `master`.

We periodically tag `-alpha` releases directly on `master`. The
semantics of such releases is that we reached a point where we have
features we would like to test using the `miniooni` research CLI
client. As part of these releases, we also update dependencies and
embedded assets. This process ensures that we perform better testing
of dependencies and assets as part of development.

The `master` branch and pull requests only run CI lightweight tests
that ensure the code still compiles, has good coverage, and we are
not introducing regressions in terms of the measurement engine.

To draft a release we branch off of `master` and create a `release/x.y`
branch where `x` is the major number and `y` is the minor number. For
release branches, we enable a very comprehensive set of tests that run
automatically with every commit. The purpose of a release branch is to
make sure all checks are green and hotfix bugs that we may discover
as part of more extensively testing a release candidate. Beta and stable
releases should occur on this branch. Subsequent patch releases should
also occur on this branch. We have one such branch for each `x.y`
release. If there are fixes on `master` that we want to backport, we
cherry-pick them into the release branch. Likewise, if we need to
forward port fixes, we cherry-pick them into `master`. When we backport,
the commit message should start with `[backport]`; when we forward
port, the commit message should start with `[forwardport]`.

When we branch off release `x.y` from `master`, we also need to bump
the `alpha` version used by `master`.

We build binary packages for each tagged release. We will use external
tools for publishing binaries to our Debian repository, Maven Central, etc.

## Community Channels

Stuck somewhere or Have any questions? please join our [Slack Channels](https://slack.ooni.org/) or [IRC](ircs://irc.oftc.net:6697/#ooni). We're here to help and always available to discuss. 

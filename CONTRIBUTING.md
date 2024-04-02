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

We use golang as our primary language for the development of OONI Probe CLI and do
check out the resources below, quite useful to read before contributing.

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

New code should have full coverage using either localhost or the
[internal/netemx](./internal/netemx/) package. Try to cover all the
error paths as well as the important properties of the code you've written
that you would like to be sure about.

Additional integration tests using the host network are good,
but they MUST use this pattern:

```Go
func TestUsingHostNetwork(t *testing.T) {
	if testing.Short() {
		t.Skip("skip test in short mode")
	}
}
```

The overall objective here is for `go test -short` to only use localhost
and [internal/netemx](./internal/netemx/) such that tests are always
reproducible. Tests using the host network are there to give us extra
confidence that everything is working as intended.

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

## Version of Go

OONI Probe release builds use a specific version Go. To make sure
you use the correct version of Go, please develop using:

```bash
./script/go.bash
```

rather than using Go directly. This script is a drop-in replacement
for the `go` command that requires Go >= 1.15, downloads the correct
version of Go in `$HOME/sdk/go1.Y.Z`, and invokes it.

By using the version of Go we'll be using for releasing, you make
sure that your contribution doesn't include functionality implemented
by later versions of Go.

## Implementation requirements

- always use `x/sys/execabs` or `./internal/shellx` instead of
using the `os/exec` package directly

- use `./internal/fsx.OpenFile` when you need to open a file

- use `./internal/netxlite.ReadAllContext` instead of `io.ReadAll`
and `./internal/netxlite.CopyContext` instead of `io.Copy`

- use `./internal/model.ErrorToStringOrOK` when
an experiment logs intermediate results

- do not call `netxlite.netxlite.NewMozillaCertPool` unless you need to
modify a copy of the default Mozilla CA pool (when using `netxlite`
as the underlying library--which is the common case--you can just
leave the `RootCAs` to `nil` in a `tls.Config` and `netxlite`
will understand you want to use the default pool)

## Code testing requirements

Make sure all tests pass with `go test ./...` run from the
top-level directory of this repository. If you're using Linux,
please, run `go test -race ./...`.

## Writing a new OONI experiment

When you are implementing a new experiment (aka nettest), make sure
you have read the relevant spec from the [ooni/spec](
https://github.com/ooni/spec) repository. If the spec is missing,
please help the pull request reviewer to create it. If the spec is
not clear, please let us know during the review.

To get a sense of what we expect from an experiment, see the [internal/tutorial](
https://github.com/ooni/probe-cli/tree/master/internal/tutorial) tutorial

## Branch management and releasing

We integrate new features in the `master` branch. If you are an external
contributor, you generally only care about that. However, if you are
part of the OONI team, you also need to care about releasing.

In terms of branching, the release process is roughly the following:

1. we use the [routine sprint releases template](
https://github.com/ooni/probe/blob/master/.github/ISSUE_TEMPLATE/routine-sprint-releases.md)
to create an issue describing the activities bound to an
upcoming OONI Probe release;

2. the first part of the procedure happens inside the `master` branch
until we reach a point where we tag an `alpha` release (e.g., `v3.21.0-alpha`);

3. once we have tagged an `alpha` release, we create and push a branch
named `release/X.Y` (e.g., `release/3.21`);

4. we commit to the `master` branch and bump the `internal/version/version.go`
version number to be the next `alpha` release, such that we can distinguish
measurements from the `master` branch taken after tagging the `alpha`;

5. we finish preparing the release and eventually tag a stable release
(e.g., `v3.21.0`) inside the `release/X.Y` branch;

6. we keep the `release/X.Y` around forever and we keep it as the
branching point from which to create patch releases (e.g., `v.3.21.1`).

The `release/X.Y` branches run many more CI checks than the `master` branch
and this allows us to ensure that everything is in order for releasing. We run
fewer checks in the `master` branch to make the development process leaner.

We prefer backporting from `master` to `release/X.Y` to forward porting from
a `release/X.Y` to `master`. When backporting, the commit name should start
with `[backport]` to identify it as a backporting commit.

## Releases

Tagging causes specific GitHub Actions to create a pre-release (if the
tag contains `-alpha` or `-beta`) or a stable release (if the tag is like
`vX.Y.Z`; e.g., `v3.21.0`).

Every night there is a GitHub Action that builds the current state of
the `master` branch and publishes it inside the [rolling release tag](
https://github.com/ooni/probe-cli/releases/tag/rolling).

We use a separate (private) repository to publish Android artefacts to
Maven Central, publish Debian packages, etc.

## Community Channels

Stuck somewhere or Have any questions? please join our
[Slack Channels](https://slack.ooni.org/) or [IRC](ircs://irc.oftc.net:6697/#ooni). We're
here to help and always available to discuss.

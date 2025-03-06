# Releasing OONI Probe CLI

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

The rest of this document discusses what you should do for each high-level
group of the [routine sprint releases template](
https://github.com/ooni/probe/blob/master/.github/ISSUE_TEMPLATE/routine-sprint-releases.md).

## Psiphon

We need to precisely pin to the Psiphon dependency by using the latest
commit in the `staging-client` branch with this command:

```bash
./script/go.bash get -u -v github.com/Psiphon-Labs/psiphon-tunnel-core@COMMIT
./script/go.bash mod tidy
./script/go.bash build -v ./...
./script/go.bash test ./...
```

Psiphon developers sometime use `replace` in their `go.mod` file. Because `replace`
only applies to the main module, we should probably ask to Psiphon developers what is
the impact of new `replace` directives on our integration.

## Go version

We MAY need to update `.github/workflows/gobash.yml` if a new minor version of Go is
available, to make sure the `./script/go.bash` tool is still able to build if run using
such a new minor version of Go. For example, I added `go1.22` after it was released.

We also typically want to update the Go version we use. The general rule is that OONI
Probe SHOULD build for any version greater than this version. We indicate the expected
Go version inside the `GOVERSION` file. We try to use a stable, still supported Go
version that is compatible with our dependencies.

We should additionally update the `toolchain` line inside of `go.mod` to use the
specific toolchain we want to use. (The `toolchain` mechanism was introduced by
Go 1.21.0 and it may be that we can now use just the `toolchain` instead of
using the `GOVERSION` file; this should eventually be investigated.)

## Android

We use `NDKVERSION` to track the expected version of the NDK to use. We should also
update this file to point to the latest stable version when releasing.

We should also update `./MOBILE/android/ensure` to update the versions of the
`build-tools`, and `platforms` that we're using for building locally.

## Dependencies other than Psiphon

Ideally, one should be able to update all dependencies using:

```bash
./script/go.bash get -u -v ./...
```

However, this MAY break the tree. For example, while writing this document, running
such a command would update to a version of uTLS that breaks Psiphon.

A less aggressive approach is that of using https://github.com/icholy/gomajor to
list the dependencies needing updating and updating each of them manually by editing
`go.mod` and then running:

```bash
./script/go.bash mod tidy
```

It's also important to update the C dependencies required to build `tor` for mobile. To this
end, just open the following files in `internal/cmd/buildtool/` and use the links to homebrew
to update the version number and the `SHA256SUM`:

- `cdepslibevent.go`
- `cdepsopenssl.go`
- `cdepstor.go`
- `cdepszlib.go`

Make sure you also update the corresponding tests in `internal/cmd/buildtool`.

At the end, before committing and pusing, one should check that it's all good using:

```bash
./script/go.bash build -v ./...
./script/go.bash test ./...
```

## Updating assets and definitions

We save some measurements results locally to test the `./internal/minipipeline` package. We
regenerate these files before releasing using this command:

```bash
./script/updateminipipeline.bash
```

We also run:

```bash
./script/go.bash generate ./...
```

to update the bundled certs.

Then, we follow the instructions in `./internal/model/http.go` to update the
`User-Agent` header that we're using when measuring.

Finally, we need to bless a new https://github.com/ooni/probe-assets release
and integrate it as a dependency. To this end, one needs to follow this procedure:

1. make sure https://github.com/ooni/historical-geoip has run and otherwise, if
the GitHub Action was paused, restart it to obtain a new database published using the
Internet Archive API on the Internet Archive cloud;

2. obtain the link to the latest database build and its SHA256;

3. follow https://github.com/ooni/probe-assets instructions in its README.md.

As usual, before committing and pushing:


```bash
./script/go.bash build -v ./...
./script/go.bash test ./...
```

## Maintenance

The check-list should be self explanatory.

## QA and alpha releasing

This stage is where we tag an `alpha` release on `master` and then create
the `release/X.Y` branch, as explained above.

## Releasing proper

This check-list stage should also be self explanatory.

## Publishing stable packages

We use `./script/autoexport.bash` in https://github.com/ooni/probe-engine to
export the latest engine to community members. Then we commit, push on master
and tag a new 0.Y.0 release. It's fine to use as target for the autoexport
the `alpha` tag or the final release tag. Alphas are pretty close to releases anyway.

For publishing for Android and Debian, head out to the
https://github.com/ooni/probe-releases private repository
and follow the `README.md` instructions.


# Directory MONOREPO

This directory contains scripts to emulate a
[monorepo](https://en.wikipedia.org/wiki/Monorepo).

## Motivation

We have two main use cases:

1. creating a [ooni/probe-android](https://github.com/ooni/probe-android)
APK using a [../pkg/oonimkall](../pkg/oonimkall/) AAR built using the
current branch of this repository;

2. creating a [ooni/probe-desktop](https://github.com/ooni/probe-desktop) release
using a [../cmd/ooniprobe](../cmd/ooniprobe) executable built using the
current branch of this repository.

We _don't need_ a monorepo to perform both activities. However, they are
manual, tedious, and error prone activities. Hence, this set of scripts
to automate the process of creating these builds.

## Local setup

If you're not a OONI developer, these scripts won't work for you because
the default configuration clones and tracks also private repositories.

To override the default configuration, run this command once:

```bash
cp -v ./MONOREPO/tools/local{.example,}.bash
```

This will create a `local.bash` file from `local.example.bash`. The newly
created file removes private repositories from the configuration.

## Architecture

The [tools](tools) directory contains the top-level tools that one should
be invoking. These are the main scripts in there:

* [gitx](tools/gitx): monorepo aware git extensions;
* [info](tools/info): prints scripts developer documentation on the stdout;
* [setupandroid](tools/setupandroid): installs the Android SDK and NDK;
* [setupgo](tools/setupgo): installs Go in `$HOME/sdk` and `$HOME/bin/go`;
* [setupshfmt](tools/setupshfmt): installs the `shfmt` tool in `$HOME/go/bin`;
* [vscode](tools/vscode): utility to open VSCode for a given repository.

The [repo](repo) directory will contain each subrepository we care about for the
purpuse of making the two use cases above possible.

The [w](w) directory contains workflows implementing use cases. These
are the most important workflows:

* [build-android-with-cli.bash](w/build-android-with-cli.bash): builds
[../pkg/oonimkall](../pkg/oonimkall/) for Android and builds
[repo/probe-android](repo/probe-android/) in experimental release
mode using the [../pkg/oonimkall](../pkg/oonimkall/) we just compiled;

* [run-desktop-with-cli.bash](w/run-desktop-with-cli.bash): builds
[../cmd/ooniprobe](../cmd/ooniprobe/) for the current system and runs
[repo/probe-desktop](repo/probe-desktop/) in debug mode using the
[../cmd/ooniprobe](../cmd/ooniprobe/) binary we just compiled.

## Monorepo aware git operations

The [gitx](tools/gitx) script contains monorepo aware git extensions
through a set of subcommands. Here's a brief overview:

* `gitx checkout {branch}` runs `git checkout {branch} || git checkout -b {branch}`
in the probe-cli repository as well as in the subrepositories. Because we run
`checkout` both without and with `-b`, the end result is that we have all
the repositories with an equally named branch. Checking out an already
existing branch helps with picking up unfinished work. Checking out a
new branch is how you start doing now work. In some cases, you are going
to have a branch already existing only for some repositories (e.g., you
started working on a feature and then realized you wanted to use the
monorepo). For this reason, we implement this mixed branch checkout strategy.

* `gitx clean` runs `git clean -dffx` in the probe-cli repository as
well as in the subrepositories making sure we don't wipe out the
checked-out subrepositories in the process.

* `gitx commit {commit-message-file}` runs `git commit -aF {commit-message-file}`
in the probe-cli repository as well as in each subrepository, thus
ensuring the whole tree state is captured by a specific commit in each repository.

* `gitx diff [flags]` runs `git diff [flags]` in the probe-cli repository
as well as in each subrepository, thus providing a whole picture of what
changed.

* `gitx push` pushes the current branch upstream for the probe-cli repository
and each subrepositories provided that there are interesting changes with
respect to the base branch. If there are no such changes, then we avoid
pushing branches to upstream unnecessarily.

* `gitx reset` calls `git reset --hard HEAD` on each repository and then
switches to the default branch of each repository. If you run `gitx reset -f`
it will additionally delete all non-default local branches.

* `gitx status` is morally equivalent to running `git status` in each
repository including the probe-cli repository.

* `gitx sync` ensures that we clone each repository, sync its default branch,
and prunes knowledge of remote branches that have been pruned also upstream.

These scripts are very opinionated in terms of how one should be
developing with git. Here is the only workflow they support:

1. you start from a clean, synced tree (`gitx clean && gitx reset -f && gitx sync`);

2. you checkout a feature branch (`gitx checkout issue/1234`);

3. you develop across the whole set of repositories;

4. you `gitx diff` to see what changed overall;

5. you use [w](w) workflows as needed;

6. you `gitx commit COMMITFILE.txt` across the whole set of repositories;

7. you `gitx push` upstream all the branches that changed;

8. you manually (for now) open pull requests;

9. you eventually merge all the PRs.

## History

The [bassosimone/monorepo](https://github.com/bassosimone/monorepo) is the
place where we've been incubating this functionality for ~one year.

#!/usr/bin/env python3

from __future__ import annotations

import getopt
import json
import os
import platform
import shlex
import subprocess
import sys

from typing import Dict
from typing import List
from typing import NoReturn
from typing import Optional
from typing import Protocol


def ANDROID_CMDLINETOOLS_OS() -> str:
    """ANDROID_CMDLINETOOLS_OS maps the name of the current OS to the
    name used by the android command-line tools file."""
    system = platform.system()
    if system == "Linux":
        return "linux"
    if system == "Darwin":
        return "mac"
    raise RuntimeError(system)


def ANDROID_CMDLINETOOLS_VERSION() -> str:
    """ANDROID_CMDLINETOOLS_VERSION returns the version of the Android
    command-line tools that we'd like to download."""
    return "6858069"


def ANDROID_CMDLINETOOLS_SHA256SUM() -> str:
    """ANDROID_CMDLINETOOLS_SHA256SUM returns the SHA256 sum of the
    Android command-line tools zip file."""
    return {
        "linux": "87f6dcf41d4e642e37ba03cb2e387a542aa0bd73cb689a9e7152aad40a6e7a08",
        "mac": "58a55d9c5bcacd7c42170d2cf2c9ae2889c6797a6128307aaf69100636f54a13",
    }[ANDROID_CMDLINETOOLS_OS()]


def CACHEDIR() -> str:
    """CACHEDIR returns the directory where we cache the SDKs."""
    return os.path.join(os.path.expandvars("${HOME}"), ".ooniprobe-build")


def GOVERSION() -> str:
    """GOVERSION is the Go version we use."""
    return "1.16.3"


def GOSHA256SUM() -> str:
    """GOSHA256SUM returns the SHA256 sum of the Go tarball."""
    return {
        "linux": {
            "amd64": "951a3c7c6ce4e56ad883f97d9db74d3d6d80d5fec77455c6ada6c1f7ac4776d2",
            "arm64": "566b1d6f17d2bc4ad5f81486f0df44f3088c3ed47a3bec4099d8ed9939e90d5d",
        },
        "darwin": {
            "amd64": "6bb1cf421f8abc2a9a4e39140b7397cdae6aca3e8d36dcff39a1a77f4f1170ac",
            "arm64": "f4e96bbcd5d2d1942f5b55d9e4ab19564da4fad192012f6d7b0b9b055ba4208f",
        },
    }[GOOS()][GOARCH()]


def GOOS() -> str:
    """GOOS returns the GOOS value for the current system."""
    system = platform.system()
    if system == "Linux":
        return "linux"
    if system == "Darwin":
        return "darwin"
    raise RuntimeError(system)


def GOARCH() -> str:
    """GOARCH returns the GOARCH value for the current system."""
    machine = platform.machine()
    if machine in ("arm64", "arm", "386", "amd64"):
        return machine
    if machine in ("x86", "i386"):
        return "386"
    if machine == "x86_64":
        return "amd64"
    if machine == "aarch64":
        return "arm64"
    raise RuntimeError(machine)


def log(msg: str) -> None:
    """log prints a message on the standard error."""
    print(msg, file=sys.stderr)


class Options(Protocol):
    """Options contains the configured options."""

    def debugging(self) -> bool:
        """debugging indicates whether to pass -x to `go build...`."""

    def dry_run(self) -> bool:
        """dry_run indicates whether to execute commands."""

    def target(self) -> str:
        """target is the target to build."""

    def verbose(self) -> bool:
        """verbose indicates whether to pass -v to `go build...`."""


class ConfigParser:
    """ConfigParser parses options from CLI flags."""

    @classmethod
    def parse(cls, targets: List[str]) -> Options:
        """parse parses command line options and returns a
        suitable configuration object."""
        conf = cls()
        conf._parse(targets)
        return conf

    def __init__(self) -> None:
        self._debugging = False
        self._dry_run = False
        self._target = ""
        self._verbose = False

    def debugging(self) -> bool:
        return self._debugging

    def dry_run(self) -> bool:
        return self._dry_run

    def target(self) -> str:
        return self._target

    def verbose(self) -> bool:
        return self._verbose

    __usage_string = """\
usage: ./make [-nvx] -t target
       ./make -l
       ./make [--help|-h]

The first form of the command builds the `target` specified using the
`-t` command line flag. The `-n` flag enables a dry run where the command
only prints the commands it would run. The `-v` and `-x` flags are
passed directly to `go build ...` and `gomobile bind ...`.

The second form of the command lists all the available targets.

The third form of the command prints this help screen.
"""

    @classmethod
    def _usage(cls, err: str = "", exitcode: int = 0) -> NoReturn:
        if err:
            sys.stderr.write("error: {}\n".format(err))
        sys.stderr.write(cls.__usage_string)
        sys.exit(exitcode)

    def _parse(self, targets: List[str]):
        try:
            opts, args = getopt.getopt(sys.argv[1:], "hlnt:vx", ["help"])
        except getopt.GetoptError as err:
            self._usage(err=err.msg, exitcode=1)
        if args:
            self._usage(err="unexpected number of positional arguments", exitcode=1)
        for key, value in opts:
            if key in ("-h", "--help"):
                self._usage()
            if key == "-l":
                sys.stdout.write("{}\n".format(json.dumps(targets, indent=4)))
                sys.exit(0)
            if key == "-n":
                self._dry_run = True
                continue
            if key == "-t":
                self._target = value
                continue
            if key == "-v":
                self._verbose = True
                continue
            if key == "-x":
                self._debugging = True
                continue
            raise RuntimeError(key, value)

        if self._target == "":  # ./build prints terse help
            self._usage()

        if self._target not in targets:
            sys.stderr.write("unknown target: {}\n".format(self._target))
            sys.stderr.write("try `./make -l` to see the available targets.\n")
            sys.exit(1)


class Engine(Protocol):
    """Engine is an engine for building targets."""

    def echo_to_file(self, content: str, filepath: str) -> None:
        """echo_to_file writes the content string to the given file."""

    def run(
        self,
        cmdline: List[str],
        cwd: Optional[str] = None,
        extra_env: Optional[Dict[str, str]] = None,
    ) -> None:
        """run runs the specified command line."""


class CommandExecutor:
    """CommandExecutor executes commands."""

    def echo_to_file(self, content: str, filepath: str) -> None:
        """echo_to_file implements Engine.echo_to_file"""
        with open(filepath, "w") as filep:
            filep.write(content)
            filep.write("\n")

    def run(
        self,
        cmdline: List[str],
        cwd: Optional[str] = None,
        extra_env: Optional[Dict[str, str]] = None,
    ) -> None:
        """run implements Engine.run."""
        env = os.environ.copy()
        if extra_env:
            for key, value in extra_env.items():
                env[key] = value
        subprocess.run(cmdline, check=True, cwd=cwd, env=env)


class CommandLogger:
    """CommandLogger logs commands."""

    def echo_to_file(self, content: str, filepath: str) -> None:
        """echo_to_file implements Engine.echo_to_file"""
        log("make: echo '{}' > {}".format(content, filepath))

    def run(
        self,
        cmdline: List[str],
        cwd: Optional[str] = None,
        extra_env: Optional[Dict[str, str]] = None,
    ) -> None:
        """run implements Engine.run."""
        cdpart = ""
        if cwd:
            cdpart = "cd {} && ".format(cwd)
        envpart = ""
        if extra_env:
            for key, value in extra_env.items():
                envpart += "{}={} ".format(key, value)
        log("make: {}{}{}".format(cdpart, envpart, shlex.join(cmdline)))


class EngineComposer:
    """EngineComposer composes two engines."""

    def __init__(self, first: Engine, second: Engine):
        self._first = first
        self._second = second

    def echo_to_file(self, content: str, filepath: str) -> None:
        """echo_to_file implements Engine.echo_to_file"""
        self._first.echo_to_file(content, filepath)
        self._second.echo_to_file(content, filepath)

    def run(
        self,
        cmdline: List[str],
        cwd: Optional[str] = None,
        extra_env: Optional[Dict[str, str]] = None,
    ) -> None:
        """run implements Engine.run."""
        self._first.run(cmdline, cwd=cwd, extra_env=extra_env)
        self._second.run(cmdline, cwd=cwd, extra_env=extra_env)


def new_engine(options: Options) -> Engine:
    """new_engine creates a new engine instance"""
    if options.dry_run():
        return CommandLogger()
    return EngineComposer(CommandLogger(), CommandExecutor())


class Target(Protocol):
    """Target is a target to build."""

    def name(self) -> str:
        """name returns the target name."""

    def build(self, engine: Engine, options: Options) -> None:
        """build builds the specified target."""


class SDKGolangGo:
    """SDKGolangGo creates ${CACHEDIR}/SDK/golang."""

    __name = os.path.join(CACHEDIR(), "SDK", "golang")

    def name(self) -> str:
        return self.__name

    def build(self, engine: Engine, options: Options) -> None:
        if os.path.isdir(self.__name):
            log("{}: already downloaded".format(self.__name))
            return
        filename = "go{}.{}-{}.tar.gz".format(GOVERSION(), GOOS(), GOARCH())
        url = "https://golang.org/dl/{}".format(filename)
        engine.run(["mkdir", "-p", self.__name])
        filepath = os.path.join(self.__name, filename)
        engine.run(["curl", "-fsSLo", filepath, url])
        sha256file = os.path.join(CACHEDIR(), "SDK", "SHA256")
        engine.echo_to_file("{}  {}".format(GOSHA256SUM(), filepath), sha256file)
        engine.run(["shasum", "--check", sha256file])
        engine.run(["rm", sha256file])
        engine.run(["tar", "-xf", filename], cwd=self.__name)
        engine.run(["rm", filepath])

    def goroot(self):
        """goroot returns the goroot."""
        return os.path.join(self.__name, "go")


class SDKOONIGo:
    """SDKOONIGo creates ${CACHEDIR}/SDK/oonigo."""

    __name = os.path.join(CACHEDIR(), "SDK", "oonigo")

    def name(self) -> str:
        return self.__name

    def build(self, engine: Engine, options: Options) -> None:
        if os.path.isdir(self.__name):
            log("{}: already downloaded".format(self.__name))
            return
        golang_go = SDKGolangGo()
        golang_go.build(engine, options)
        engine.run(
            [
                "git",
                "clone",
                "-b",
                "ooni",
                "--single-branch",
                "--depth",
                "8",
                "https://github.com/ooni/go",
                self.__name,
            ]
        )
        engine.run(
            ["./make.bash"],
            cwd=os.path.join(self.__name, "src"),
            extra_env={"GOROOT_BOOTSTRAP": golang_go.goroot()},
        )


class AndroidSDK:
    """AndroidSDK creates ${CACHEDIR}/SDK/android."""

    __name = os.path.join(CACHEDIR(), "SDK", "android")

    def name(self) -> str:
        return self.__name

    def build(self, engine: Engine, options: Options) -> None:
        if os.path.isdir(self.__name):
            log("{}: already downloaded".format(self.__name))
            return
        filename = "commandlinetools-{}-{}_latest.zip".format(
            ANDROID_CMDLINETOOLS_OS(), ANDROID_CMDLINETOOLS_VERSION()
        )
        url = "https://dl.google.com/android/repository/{}".format(filename)
        engine.run(["mkdir", "-p", self.__name])
        filepath = os.path.join(self.__name, filename)
        engine.run(["curl", "-fsSLo", filepath, url])
        sha256file = os.path.join(CACHEDIR(), "SDK", "SHA256")
        engine.echo_to_file(
            "{}  {}".format(ANDROID_CMDLINETOOLS_SHA256SUM(), filepath), sha256file
        )
        engine.run(["shasum", "--check", sha256file])
        engine.run(["rm", sha256file])
        engine.run(["unzip", filename], cwd=self.__name)
        engine.run(["rm", filepath])
        engine.run(
            ["mv", "cmdline-tools", ANDROID_CMDLINETOOLS_VERSION()], cwd=self.__name
        )
        engine.run(["mkdir", "cmdline-tools"], cwd=self.__name)
        engine.run(
            ["mv", ANDROID_CMDLINETOOLS_VERSION(), "cmdline-tools"], cwd=self.__name
        )


TARGETS: List[Target] = [
    AndroidSDK(),
    SDKGolangGo(),
    SDKOONIGo(),
]


def main() -> None:
    """main function"""
    alltargets: Dict[str, Target] = dict((t.name(), t) for t in TARGETS)
    options = ConfigParser.parse(list(alltargets.keys()))
    engine = new_engine(options)
    selected = alltargets[options.target()]
    selected.build(engine, options)


if __name__ == "__main__":
    main()
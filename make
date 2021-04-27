#!/usr/bin/env python3

""" Build script for ooniprobe. You can get documentation regarding
its usage by running `./make --help`. """

from __future__ import annotations
import datetime

import getopt
import json
import os
import platform
import shlex
import shutil
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


def GOPATH() -> str:
    """GOPATH is the GOPATH we use."""
    return os.path.expandvars("${HOME}/go")


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


def ANDROID_NDK_VERSION() -> str:
    """ANDROID_NDK_VERSION returns the Android NDK version."""
    return "22.1.7171670"


def SDKMANAGER_INSTALL_CMD(binpath: str) -> List[str]:
    """SDKMANAGER_INSTALL_CMD returns the command line for installing
    all the required dependencies using the sdkmanager."""
    return [
        os.path.join(binpath, "sdkmanager"),
        "--install",
        "build-tools;29.0.3",
        "platforms;android-30",
        "ndk;{}".format(ANDROID_NDK_VERSION()),
    ]


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
    def parse(cls, targets: List[str]) -> ConfigParser:
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

    # The main reason why I am using getopt here is such that I am able
    # to print a very clear and detailed usage string. (But the same
    # could be obtained quite likely w/ argparse.)

    __usage_string = """\
usage: ./make [-nvx] -t target
       ./make -l
       ./make [--help|-h]

The first form of the command builds the `target` specified using the
`-t` command line flag. If the target has dependencies, this command will
build the dependent targets first. The `-n` flag enables a dry run where
the command only prints the commands it would run. The `-v` and `-x` flags
are passed directly to `go build ...` and `gomobile bind ...`.

The second form of the command lists all the available targets as
a pretty-printed JSON list.

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

        if self._target == "":  # no arguments is equivalent to --help
            self._usage()

        if self._target not in targets:
            sys.stderr.write("unknown target: {}\n".format(self._target))
            sys.stderr.write("try `./make -l` to see the available targets.\n")
            sys.exit(1)


class Engine(Protocol):
    """Engine is an engine for building targets."""

    def cat_sed_redirect(
        self, pattern: str, value: str, source: str, dest: str
    ) -> None:
        """cat_sed_redirect does `cat $source|sed "s/$pattern/$value/g" > $dest`."""

    def echo_to_file(self, content: str, filepath: str) -> None:
        """echo_to_file writes the content string to the given file."""

    def require(self, *executable: str) -> None:
        """require fails if executable is not found in path."""

    def run(
        self,
        cmdline: List[str],
        cwd: Optional[str] = None,
        extra_env: Optional[Dict[str, str]] = None,
        inputbytes: Optional[bytes] = None,
    ) -> None:
        """run runs the specified command line."""


class CommandRealExecutor:
    """CommandRealExecutor executes commands."""

    def cat_sed_redirect(
        self, pattern: str, value: str, source: str, dest: str
    ) -> None:
        """cat_sed_redirect implements Engine.cat_sed_redirect."""
        with open(source, "r") as sourcefp:
            data = sourcefp.read().replace(pattern, value)
            with open(dest, "w") as destfp:
                destfp.write(data)

    def echo_to_file(self, content: str, filepath: str) -> None:
        """echo_to_file implements Engine.echo_to_file"""
        with open(filepath, "w") as filep:
            filep.write(content)
            filep.write("\n")

    def require(self, *executable: str) -> None:
        """require implements Engine.require."""

    def run(
        self,
        cmdline: List[str],
        cwd: Optional[str] = None,
        extra_env: Optional[Dict[str, str]] = None,
        inputbytes: Optional[bytes] = None,
    ) -> None:
        """run implements Engine.run."""
        env = os.environ.copy()
        if extra_env:
            for key, value in extra_env.items():
                env[key] = value
        subprocess.run(cmdline, check=True, cwd=cwd, env=env, input=inputbytes)


class CommandDryRunner:
    """CommandDryRunner is the dry runner."""

    # Implementation note: here we try to log valid bash snippets
    # such that is really obvious what we are doing.

    def __init__(self):
        self.__cmdcache: Dict[str, str] = {}

    def cat_sed_redirect(
        self, pattern: str, value: str, source: str, dest: str
    ) -> None:
        """cat_sed_redirect implements Engine.cat_sed_redirect."""
        log('./make: cat {}|sed "s/{}/{}/g" > {}'.format(source, pattern, value, dest))

    def echo_to_file(self, content: str, filepath: str) -> None:
        """echo_to_file implements Engine.echo_to_file"""
        log("./make: echo '{}' > {}".format(content, filepath))

    def require(self, *executable: str) -> None:
        """require implements Engine.require."""
        for exc in executable:
            if exc in self.__cmdcache:
                continue  # do not print checks more than once
            fullpath = shutil.which(exc)
            if not fullpath:
                log("./make: checking for {}... not found".format(exc))
                sys.exit(1)
            log("./make: checking for {}... {}".format(exc, fullpath))
            self.__cmdcache[exc] = fullpath

    def run(
        self,
        cmdline: List[str],
        cwd: Optional[str] = None,
        extra_env: Optional[Dict[str, str]] = None,
        inputbytes: Optional[bytes] = None,
    ) -> None:
        """run implements Engine.run."""
        cdpart = ""
        if cwd:
            cdpart = "cd {} && ".format(cwd)
        envpart = ""
        if extra_env:
            for key, value in extra_env.items():
                envpart += shlex.join(["{}={}".format(key, value)]) + " "
        log("./make: {}{}{}".format(cdpart, envpart, shlex.join(cmdline)))


class EngineComposer:
    """EngineComposer composes two engines."""

    def __init__(self, first: Engine, second: Engine):
        self._first = first
        self._second = second

    def cat_sed_redirect(
        self, pattern: str, value: str, source: str, dest: str
    ) -> None:
        """cat_sed_redirect implements Engine.cat_sed_redirect."""
        self._first.cat_sed_redirect(pattern, value, source, dest)
        self._second.cat_sed_redirect(pattern, value, source, dest)

    def echo_to_file(self, content: str, filepath: str) -> None:
        """echo_to_file implements Engine.echo_to_file"""
        self._first.echo_to_file(content, filepath)
        self._second.echo_to_file(content, filepath)

    def require(self, *executable: str) -> None:
        """require implements Engine.require."""
        self._first.require(*executable)
        self._second.require(*executable)

    def run(
        self,
        cmdline: List[str],
        cwd: Optional[str] = None,
        extra_env: Optional[Dict[str, str]] = None,
        inputbytes: Optional[bytes] = None,
    ) -> None:
        """run implements Engine.run."""
        self._first.run(cmdline, cwd=cwd, extra_env=extra_env, inputbytes=inputbytes)
        self._second.run(cmdline, cwd=cwd, extra_env=extra_env, inputbytes=inputbytes)


def new_engine(options: Options) -> Engine:
    """new_engine creates a new engine instance"""
    if options.dry_run():
        return CommandDryRunner()
    return EngineComposer(CommandDryRunner(), CommandRealExecutor())


class Target(Protocol):
    """Target is a target to build."""

    def name(self) -> str:
        """name returns the target name."""

    def build(self, engine: Engine, options: Options) -> None:
        """build builds the specified target."""


class SDKGolangGo:
    """SDKGolangGo creates ${CACHEDIR}/SDK/golang."""

    # We download a golang SDK from upstream to make sure we
    # are always using a specific version of golang/go.

    __name = os.path.join(CACHEDIR(), "SDK", "golang")

    def name(self) -> str:
        return self.__name

    def build(self, engine: Engine, options: Options) -> None:
        if os.path.isdir(self.__name) and not options.dry_run():
            log("./make: {}: already built".format(self.__name))
            return
        log("./make: building {}...".format(self.__name))
        engine.require("mkdir", "curl", "shasum", "rm", "tar", "echo")
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

    # We use a private fork of golang/go on Android as a
    # workaround for https://github.com/ooni/probe/issues/1444

    __name = os.path.join(CACHEDIR(), "SDK", "oonigo")

    def name(self) -> str:
        return self.__name

    def binpath(self) -> str:
        return os.path.join(self.__name, "bin")

    def build(self, engine: Engine, options: Options) -> None:
        if os.path.isdir(self.__name) and not options.dry_run():
            log("./make: {}: already built".format(self.__name))
            return
        golang_go = SDKGolangGo()
        golang_go.build(engine, options)
        log("./make: building {}...".format(self.__name))
        engine.require("git", "bash")
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


class SDKAndroid:
    """SDKAndroid creates ${CACHEDIR}/SDK/android."""

    __name = os.path.join(CACHEDIR(), "SDK", "android")

    def name(self) -> str:
        return self.__name

    def home(self) -> str:
        return self.__name

    def ndk_home(self) -> str:
        return os.path.join(self.home(), "ndk", ANDROID_NDK_VERSION())

    def build(self, engine: Engine, options: Options) -> None:
        if os.path.isdir(self.__name) and not options.dry_run():
            log("./make: {}: already built".format(self.__name))
            return
        log("./make: building {}...".format(self.__name))
        engine.require("mkdir", "curl", "echo", "shasum", "rm", "unzip", "mv", "java")
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
        engine.run(
            SDKMANAGER_INSTALL_CMD(
                os.path.join(
                    self.__name,
                    "cmdline-tools",
                    ANDROID_CMDLINETOOLS_VERSION(),
                    "bin",
                ),
            ),
            inputbytes=b"Y\n",  # automatically accept license
        )


class OONIProbePrivate:
    """OONIProbePrivate creates ${CACHEDIR}/github.com/ooni/probe-private."""

    # We use this private repository to copy the psiphon configuration
    # file to embed into the ooniprobe binaries

    __name = os.path.join(CACHEDIR(), "github.com", "ooni", "probe-private")

    def name(self) -> str:
        return self.__name

    def copyfiles(self, engine: Engine) -> None:
        """copyfiles copies psiphon config to the repository."""
        engine.run(
            [
                "cp",
                os.path.join(self.__name, "psiphon-config.json.age"),
                os.path.join("internal", "engine"),
            ]
        )
        engine.run(
            [
                "cp",
                os.path.join(self.__name, "psiphon-config.key"),
                os.path.join("internal", "engine"),
            ]
        )

    def build(self, engine: Engine, options: Options) -> None:
        if os.path.isdir(self.__name) and not options.dry_run():
            log("./make: {}: already built".format(self.__name))
            return
        log("./make: building {}...".format(self.__name))
        engine.require("git", "cp")
        engine.run(
            [
                "git",
                "clone",
                "git@github.com:ooni/probe-private",
                self.__name,
            ]
        )


class OONIMKAllAAR:
    """OONIMKAllAAR creates ./MOBILE/android/oonimkall.aar."""

    __name = os.path.join(".", "MOBILE", "android", "oonimkall.aar")

    def name(self) -> str:
        return self.__name

    def aarfile(self) -> str:
        return self.__name

    def srcfile(self) -> str:
        return os.path.join(".", "MOBILE", "android", "oonimkall-sources.jar")

    def build(self, engine: Engine, options: Options) -> None:
        if os.path.isfile(self.__name) and not options.dry_run():
            log("./make: {}: already built".format(self.__name))
            return
        ooprivate = OONIProbePrivate()
        ooprivate.build(engine, options)
        oonigo = SDKOONIGo()
        oonigo.build(engine, options)
        android = SDKAndroid()
        android.build(engine, options)
        log("./make: building {}...".format(self.__name))
        ooprivate.copyfiles(engine)
        engine.require("sh", "javac")
        self._go_get_gomobile(engine, options, oonigo)
        self._gomobile_init(engine, oonigo, android)
        self._gomobile_bind(engine, options, oonigo, android)

    # Implementation note: we use proxy scripts for go and gomobile
    # that explicitly print what they resolve go and gomobile to using
    # `command -v`. This gives us extra confidence that we are really
    # using the oonigo fork of golang/go.

    def _go_get_gomobile(
        self, engine: Engine, options: Options, oonigo: SDKOONIGo
    ) -> None:
        # TODO(bassosimone): find a way to run this command without
        # adding extra dependencies to go.mod and go.sum.
        cmdline: List[str] = []
        cmdline.append(os.path.join(".", "MOBILE", "android", "go"))
        cmdline.append("get")
        cmdline.append("-u")
        if options.verbose():
            cmdline.append("-v")
        if options.debugging():
            cmdline.append("-x")
        cmdline.append("golang.org/x/mobile/cmd/gomobile@latest")
        engine.run(
            cmdline,
            extra_env={
                "PATH": os.pathsep.join(
                    [
                        oonigo.binpath(),  # so we use our go fork
                        os.environ["PATH"],  # original path
                    ]
                ),
                "GOPATH": GOPATH(),  # where to install gomobile
            },
        )

    def _gomobile_init(
        self,
        engine: Engine,
        oonigo: SDKOONIGo,
        android: SDKAndroid,
    ) -> None:
        cmdline: List[str] = []
        cmdline.append(os.path.join(".", "MOBILE", "android", "gomobile"))
        cmdline.append("init")
        engine.run(
            cmdline,
            extra_env={
                "PATH": os.pathsep.join(
                    [
                        os.path.join(GOPATH(), "bin"),  # for gomobile
                        oonigo.binpath(),  # for our go fork
                        os.environ["PATH"],  # original environment
                    ]
                ),
                "ANDROID_HOME": android.home(),
                "ANDROID_NDK_HOME": android.ndk_home(),
            },
        )

    def _gomobile_bind(
        self,
        engine: Engine,
        options: Options,
        oonigo: SDKOONIGo,
        android: SDKAndroid,
    ) -> None:
        cmdline: List[str] = []
        cmdline.append(os.path.join(".", "MOBILE", "android", "gomobile"))
        cmdline.append("bind")
        if options.verbose():
            cmdline.append("-v")
        if options.debugging():
            cmdline.append("-x")
        cmdline.append("-target")
        cmdline.append("android")
        cmdline.append("-o")
        cmdline.append(self.__name)
        cmdline.append("-tags")
        cmdline.append("ooni_psiphon_config")
        cmdline.append("-ldflags")
        cmdline.append("-s -w")
        cmdline.append("./pkg/oonimkall")
        engine.run(
            cmdline,
            extra_env={
                "PATH": os.pathsep.join(
                    [
                        os.path.join(GOPATH(), "bin"),  # for gomobile
                        oonigo.binpath(),  # for our go fork
                        os.environ["PATH"],  # original environment
                    ]
                ),
                "ANDROID_HOME": android.home(),
                "ANDROID_NDK_HOME": android.ndk_home(),
            },
        )


class BundleJAR:
    """BundleJAR creates ./MOBILE/android/bundle.jar."""

    # We upload the bundle.jar file to maven central to bless
    # a new release of the OONI libraries for Android.

    __name = os.path.join(".", "MOBILE", "android", "bundle.jar")

    def name(self) -> str:
        return self.__name

    def build(self, engine: Engine, options: Options) -> None:
        if os.path.isfile(self.__name) and not options.dry_run():
            log("./make: {}: already built".format(self.__name))
            return
        oonimkall = OONIMKAllAAR()
        oonimkall.build(engine, options)
        log("./make: building {}...".format(self.__name))
        engine.require("cp", "gpg", "jar")
        version = datetime.datetime.now().strftime("%Y.%m.%d-%H%M%S")
        engine.run(
            [
                "cp",
                oonimkall.aarfile(),
                os.path.join("MOBILE", "android", "oonimkall-{}.aar".format(version)),
            ]
        )
        engine.run(
            [
                "cp",
                oonimkall.srcfile(),
                os.path.join(
                    "MOBILE", "android", "oonimkall-{}-sources.jar".format(version)
                ),
            ]
        )
        engine.cat_sed_redirect(
            "@VERSION@",
            version,
            os.path.join("MOBILE", "template.pom"),
            os.path.join("MOBILE", "android", "oonimkall-{}.pom".format(version)),
        )
        names = (
            "oonimkall-{}.aar".format(version),
            "oonimkall-{}-sources.jar".format(version),
            "oonimkall-{}.pom".format(version),
        )
        for name in names:
            engine.run(
                [
                    "gpg",
                    "-abu",
                    "simone@openobservatory.org",
                    name,
                ],
                cwd=os.path.join(".", "MOBILE", "android"),
            )
        allnames = [name + ".asc" for name in names]
        allnames.extend(names)
        engine.run(
            [
                "jar",
                "-cf",
                "bundle.jar",
                *allnames,
            ],
            cwd=os.path.join(".", "MOBILE", "android"),
        )


class Android:
    """Android is the toplevel android target."""

    def name(self) -> str:
        return "android"

    def build(self, engine: Engine, options: Options) -> None:
        bundlejar = BundleJAR()
        bundlejar.build(engine, options)


TARGETS: List[Target] = [
    Android(),
    BundleJAR(),
    OONIMKAllAAR(),
    OONIProbePrivate(),
    SDKAndroid(),
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

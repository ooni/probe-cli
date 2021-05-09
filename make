#!/usr/bin/env python3

""" Build script for ooniprobe. You can get documentation regarding
its usage by running `./make --help`. """

from __future__ import annotations
import datetime

import getopt
import os
import platform
import shlex
import shutil
import subprocess
import sys

from typing import Any
from typing import Callable
from typing import Dict
from typing import List
from typing import NoReturn
from typing import Optional
from typing import Protocol
from typing import Tuple


def android_cmdlinetools_os() -> str:
    """android_cmdlinetools_os maps the name of the current OS to the
    name used by the android command-line tools file."""
    system = platform.system()
    if system == "Linux":
        return "linux"
    if system == "Darwin":
        return "mac"
    raise RuntimeError(system)


def android_cmdlinetools_version() -> str:
    """android_cmdlinetools_version returns the version of the Android
    command-line tools that we'd like to download."""
    return "6858069"


def android_cmdlinetools_sha256sum() -> str:
    """android_cmdlinetools_sha256sum returns the SHA256 sum of the
    Android command-line tools zip file."""
    return {
        "linux": "87f6dcf41d4e642e37ba03cb2e387a542aa0bd73cb689a9e7152aad40a6e7a08",
        "mac": "58a55d9c5bcacd7c42170d2cf2c9ae2889c6797a6128307aaf69100636f54a13",
    }[android_cmdlinetools_os()]


def cachedir() -> str:
    """cachedir returns the directory where we cache the SDKs."""
    return os.path.join(os.path.expandvars("${HOME}"), ".ooniprobe-build")


def goversion() -> str:
    """goversion is the Go version we use."""
    return "1.16.4"


def gopath() -> str:
    """gopath is the GOPATH we use."""
    return os.path.expandvars("${HOME}/go")


def android_ndk_version() -> str:
    """android_ndk_version returns the Android NDK version."""
    return "22.1.7171670"


def sdkmanager_install_cmd(binpath: str) -> List[str]:
    """sdkmanager_install_cmd returns the command line for installing
    all the required dependencies using the sdkmanager."""
    return [
        os.path.join(binpath, "sdkmanager"),
        "--install",
        "build-tools;29.0.3",
        "platforms;android-30",
        "ndk;{}".format(android_ndk_version()),
    ]


def log(msg: str) -> None:
    """log prints a message on the standard output."""
    print(msg, flush=True)


class Options(Protocol):
    """Options contains the configured options."""

    def debugging(self) -> bool:
        """debugging indicates whether to pass -x to `go build...`."""

    def disable_embedding_psiphon_config(self) -> bool:
        """disable_embedding_psiphon_config indicates that the user
        does not want us to embed an encrypted psiphon config file into
        the binary."""

    def dry_run(self) -> bool:
        """dry_run indicates whether to execute commands."""

    def target(self) -> str:
        """target is the target to build."""

    def verbose(self) -> bool:
        """verbose indicates whether to pass -v to `go build...`."""


class ConfigFromCLI:
    """ConfigFromCLI parses options from CLI flags."""

    @classmethod
    def parse(cls, targets: List[str], top_targets: List[Target]) -> ConfigFromCLI:
        """parse parses command line options and returns a
        suitable configuration object."""
        conf = cls()
        conf._parse(targets, top_targets)
        return conf

    def __init__(self) -> None:
        self._debugging = False
        self._disable_embedding_psiphon_config = False
        self._dry_run = False
        self._target = ""
        self._verbose = False

    def debugging(self) -> bool:
        return self._debugging

    def disable_embedding_psiphon_config(self) -> bool:
        return self._disable_embedding_psiphon_config

    def dry_run(self) -> bool:
        return self._dry_run

    def target(self) -> str:
        return self._target

    def verbose(self) -> bool:
        return self._verbose

    # The main reason why I am using getopt here is such that I am able
    # to print a very clear and detailed usage string. (But the same
    # could be obtained quite likely w/ argparse.)

    _usage_string = """\
usage: ./make [--disable-embedding-psiphon-config] [-nvx] -t target
       ./make -l
       ./make [--help|-h]

The first form of the command builds the `target` specified using the
`-t` command line flag. If the target has dependencies, this command will
build the dependent targets first. The `-n` flag enables a dry run where
the command only prints the commands it would run. The `-v` and `-x` flags
are passed directly to `go build ...` and `gomobile bind ...`. The
`--disable-embedding-psiphon-config` flag causes this command to disable
embedding a psiphon config file into the generated binary; you should
use this option when you cannot clone the private repository containing
the psiphon configuration file.

The second form of the command lists all the available targets, showing
top-level targets and their recursive dependencies.

The third form of the command prints this help screen.
"""

    @classmethod
    def _usage(cls, err: str = "", exitcode: int = 0) -> NoReturn:
        if err:
            sys.stderr.write("error: {}\n".format(err))
        sys.stderr.write(cls._usage_string)
        sys.exit(exitcode)

    def _parse(self, targets: List[str], top_targets: List[Target]):
        try:
            opts, args = getopt.getopt(
                sys.argv[1:], "hlnt:vx", ["disable-embedding-psiphon-config", "help"]
            )
        except getopt.GetoptError as err:
            self._usage(err=err.msg, exitcode=1)
        if args:
            self._usage(err="unexpected number of positional arguments", exitcode=1)
        for key, value in opts:
            if key == "--disable-embedding-psiphon-config":
                self._disable_embedding_psiphon_config = True
                continue
            if key in ("-h", "--help"):
                self._usage()
            if key == "-l":
                self._list_targets(top_targets)
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

    def _list_targets(self, top_targets: List[Target]) -> NoReturn:
        for tgt in top_targets:
            self._print_target(tgt, 0)
        sys.exit(0)

    def _print_target(self, target: Target, indent: int) -> None:
        sys.stdout.write(
            "{}{}{}\n".format(
                "  " * indent, target.name(), ":" if target.deps() else ""
            )
        )
        for dep in target.deps():
            self._print_target(dep, indent + 1)
        if indent <= 0:
            sys.stdout.write("\n")


class Engine(Protocol):
    """Engine is an engine for building targets."""

    def backticks(self, cmdline: List[str]) -> str:
        """backticks executes the given command line and returns
        the output emitted by the command to the caller."""

    def cat_sed_redirect(
        self, patterns: List[Tuple[str, str]], source: str, dest: str
    ) -> None:
        """cat_sed_redirect does
        `cat $source|sed -e "s/$patterns[0][0]/$patterns[0][1]/g" ... > $dest`."""

    def echo_to_file(self, content: str, filepath: str) -> None:
        """echo_to_file writes the content string to the given file."""

    def require(self, *executable: str) -> None:
        """require fails if executable is not found in path."""

    def run(
        self,
        cmdline: List[str],
        cwd: Optional[str] = None,
        inputbytes: Optional[bytes] = None,
    ) -> None:
        """run runs the specified command line."""

    def setenv(self, key: str, value: str) -> Optional[str]:
        """setenv sets an environment variable and returns the
        previous value of such variable (or None)."""

    def unsetenv(self, key: str) -> None:
        """unsetenv clears an environment variable."""


class CommandExecutor:
    """CommandExecutor executes commands."""

    def __init__(self, dry_runner: Engine):
        self._dry_runner = dry_runner

    def backticks(self, cmdline: List[str]) -> str:
        """backticks implements Engine.backticks"""
        # Nothing else to do, because backticks is fully
        # implemented by CommandDryRunner.
        return self._dry_runner.backticks(cmdline)

    def cat_sed_redirect(
        self, patterns: List[Tuple[str, str]], source: str, dest: str
    ) -> None:
        """cat_sed_redirect implements Engine.cat_sed_redirect."""
        self._dry_runner.cat_sed_redirect(patterns, source, dest)
        with open(source, "r") as sourcefp:
            data = sourcefp.read()
            for p, v in patterns:
                data = data.replace(p, v)
            with open(dest, "w") as destfp:
                destfp.write(data)

    def echo_to_file(self, content: str, filepath: str) -> None:
        """echo_to_file implements Engine.echo_to_file"""
        self._dry_runner.echo_to_file(content, filepath)
        with open(filepath, "w") as filep:
            filep.write(content)
            filep.write("\n")

    def require(self, *executable: str) -> None:
        """require implements Engine.require."""
        for exc in executable:
            self._dry_runner.require(exc)
            fullpath = shutil.which(exc)
            if not fullpath:
                log("checking for {}... not found".format(exc))
                sys.exit(1)
            log("checking for {}... {}".format(exc, fullpath))

    def run(
        self,
        cmdline: List[str],
        cwd: Optional[str] = None,
        inputbytes: Optional[bytes] = None,
    ) -> None:
        """run implements Engine.run."""
        self._dry_runner.run(cmdline, cwd, inputbytes)
        subprocess.run(cmdline, check=True, cwd=cwd, input=inputbytes)

    def setenv(self, key: str, value: str) -> Optional[str]:
        """setenv implements Engine.setenv."""
        # Nothing else to do, because setenv is fully
        # implemented by CommandDryRunner.
        return self._dry_runner.setenv(key, value)

    def unsetenv(self, key: str) -> None:
        """unsetenv implements Engine.unsetenv."""
        # Nothing else to do, because unsetenv is fully
        # implemented by CommandDryRunner.
        self._dry_runner.unsetenv(key)


class CommandDryRunner:
    """CommandDryRunner is the dry runner."""

    # Implementation note: here we try to log valid bash snippets
    # such that is really obvious what we are doing.

    def backticks(self, cmdline: List[str]) -> str:
        """backticks implements Engine.backticks"""
        # The backticks command is used to gather information used by
        # other commands. As such, it needs to always run. If it was not
        # running, we could not correctly implement the `-n` flag. It's
        # also a silent command, because it's not really part of the
        # sequence of bash commands that are executed. ¯\_(ツ)_/¯
        popen = subprocess.Popen(cmdline, stdout=subprocess.PIPE)
        stdout = popen.communicate()[0]
        if popen.returncode != 0:
            raise RuntimeError(popen.returncode)
        return stdout.decode("utf-8").strip()

    def cat_sed_redirect(
        self, patterns: List[Tuple[str, str]], source: str, dest: str
    ) -> None:
        """cat_sed_redirect implements Engine.cat_sed_redirect."""
        out = "./make: cat {}|sed".format(source)
        for p, v in patterns:
            out += " -e 's/{}/{}/g'".format(p, v)
        out += " > {}".format(dest)
        log(out)

    def echo_to_file(self, content: str, filepath: str) -> None:
        """echo_to_file implements Engine.echo_to_file"""
        log("./make: echo '{}' > {}".format(content, filepath))

    def require(self, *executable: str) -> None:
        """require implements Engine.require."""
        for exc in executable:
            log(f"./make: echo -n 'checking for {exc}... '")
            log("./make: command -v %s || { echo 'not found'; exit 1 }" % exc)

    def run(
        self,
        cmdline: List[str],
        cwd: Optional[str] = None,
        inputbytes: Optional[bytes] = None,
    ) -> None:
        """run implements Engine.run."""
        cdpart = ""
        if cwd:
            cdpart = "cd {} && ".format(cwd)
        log("./make: {}{}".format(cdpart, shlex.join(cmdline)))

    def setenv(self, key: str, value: str) -> Optional[str]:
        """setenv implements Engine.setenv."""
        log("./make: export {}={}".format(key, shlex.join([value])))
        prev = os.environ.get(key)
        os.environ[key] = value
        return prev

    def unsetenv(self, key: str) -> None:
        """unsetenv implements Engine.unsetenv."""
        log("./make: unset {}".format(key))
        del os.environ[key]


def new_engine(options: Options) -> Engine:
    """new_engine creates a new engine instance"""
    out: Engine = CommandDryRunner()
    if not options.dry_run():
        out = CommandExecutor(out)
    return out


class Environ:
    """Environ creates a context where specific environment
    variables are set. They will be restored to their previous
    value when we are leaving the context."""

    def __init__(self, engine: Engine, key: str, value: str) -> None:
        self._engine = engine
        self._key = key
        self._value = value
        self._prev: Optional[str] = None

    def __enter__(self) -> None:
        self._prev = self._engine.setenv(self._key, self._value)

    def __exit__(self, type: Any, value: Any, traceback: Any) -> bool:
        if self._prev is None:
            self._engine.unsetenv(self._key)
            return False  # propagate exc
        self._engine.setenv(self._key, self._prev)
        return False  # propagate exc


class AugmentedPath(Environ):
    """AugementedPath is an Environ that prepends the required
    directory to the currently existing search path."""

    def __init__(self, engine: Engine, directory: str):
        value = os.pathsep.join([directory, os.environ["PATH"]])
        super().__init__(engine, "PATH", value)


class WorkingDir:
    """WorkingDir is a context manager that enters into a given working
    directory and returns to the previous directory when done."""

    def __init__(self, dirpath: str) -> None:
        self._dirpath = dirpath
        self._prev: str = ""

    def __enter__(self) -> None:
        self._prev = os.getcwd()
        os.chdir(self._dirpath)

    def __exit__(self, type: Any, value: Any, traceback: Any) -> bool:
        os.chdir(self._prev)
        return False  # propagate exc


class Target(Protocol):
    """Target is a target to build."""

    def name(self) -> str:
        """name returns the target name."""

    def build(self, engine: Engine, options: Options) -> None:
        """build builds the specified target."""

    def deps(self) -> List[Target]:
        """deps returns the target dependencies."""


class BaseTarget:
    """BaseTarget is the base class of all targets."""

    def __init__(self, name: str, deps: List[Target]) -> None:
        # prevent child classes from easily using these variables
        self.__name = name
        self.__deps = deps

    def name(self) -> str:
        """name implements Target.name"""
        return self.__name

    def build_child_targets(self, engine: Engine, options: Options) -> None:
        """build_child_targets builds all the child targets"""
        for dep in self.__deps:
            dep.build(engine, options)

    def deps(self) -> List[Target]:
        """deps implements Target.deps."""
        return self.__deps


class SDKGolangGo(BaseTarget):
    """SDKGolangGo creates ${cachedir}/SDK/golang."""

    # We download a golang SDK from upstream to make sure we
    # are always using a specific version of golang/go.
    #
    # TODO(bassosimone): find a way to avoid polluting the
    # go.mod and go.sum when running this command.

    def __init__(self) -> None:
        name = os.path.join(
            os.path.expandvars("$HOME"), "sdk", "go{}".format(goversion())
        )
        super().__init__(name, [])

    def binpath(self) -> str:
        """binpath returns the path where the go binary is installed."""
        return os.path.join(self.name(), "bin")

    def build(self, engine: Engine, options: Options) -> None:
        """build implements Target.build"""
        if os.path.isdir(self.name()) and not options.dry_run():
            log("\n./make: {}: already built".format(self.name()))
            return
        self.build_child_targets(engine, options)
        log("\n./make: building {}...".format(self.name()))
        engine.require("go")
        cmdline: List[str] = []
        cmdline.append("go")
        cmdline.append("get")
        if options.debugging():
            cmdline.append("-x")
        if options.verbose():
            cmdline.append("-v")
        cmdline.append("golang.org/dl/go{}".format(goversion()))
        engine.run(cmdline)
        with AugmentedPath(engine, os.path.join(gopath(), "bin")):
            engine.run(["go{}".format(goversion()), "download"])
        gobin = os.path.join(self.binpath(), "go")
        if not os.path.isfile(gobin):
            sys.exit("fatal: cannot find {} file".format(gobin))
        if not os.access(gobin, os.X_OK):
            sys.exit("fatal: {} file is not executable".format(gobin))

    def goroot(self):
        """goroot returns the goroot."""
        return os.path.join(self.name(), "go")


class SDKOONIGo(BaseTarget):
    """SDKOONIGo creates ${cachedir}/SDK/oonigo."""

    # We use a private fork of golang/go on Android as a
    # workaround for https://github.com/ooni/probe/issues/1444

    def __init__(self) -> None:
        name = os.path.join(cachedir(), "SDK", "oonigo")
        self._gogo = SDKGolangGo()
        super().__init__(name, [self._gogo])

    def binpath(self) -> str:
        """binpath returns the path where the go binary is installed."""
        return os.path.join(self.name(), "bin")

    def build(self, engine: Engine, options: Options) -> None:
        """build implements Target.build"""
        if os.path.isdir(self.name()) and not options.dry_run():
            log("\n./make: {}: already built".format(self.name()))
            return
        self.build_child_targets(engine, options)
        log("\n./make: building {}...".format(self.name()))
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
                self.name(),
            ]
        )
        with Environ(engine, "GOROOT_BOOTSTRAP", self._gogo.goroot()):
            engine.run(
                ["./make.bash"],
                cwd=os.path.join(self.name(), "src"),
            )


class SDKAndroid(BaseTarget):
    """SDKAndroid creates ${cachedir}/SDK/android."""

    def __init__(self) -> None:
        name = os.path.join(cachedir(), "SDK", "android")
        super().__init__(name, [])

    def home(self) -> str:
        """home returns the ANDROID_HOME"""
        return self.name()

    def ndk_home(self) -> str:
        """ndk_home returns the ANDROID_NDK_HOME"""
        return os.path.join(self.home(), "ndk", android_ndk_version())

    def build(self, engine: Engine, options: Options) -> None:
        """build implements Target.build"""
        if os.path.isdir(self.name()) and not options.dry_run():
            log("\n./make: {}: already built".format(self.name()))
            return
        log("\n./make: building {}...".format(self.name()))
        engine.require("mkdir", "curl", "echo", "shasum", "rm", "unzip", "mv", "java")
        filename = "commandlinetools-{}-{}_latest.zip".format(
            android_cmdlinetools_os(), android_cmdlinetools_version()
        )
        url = "https://dl.google.com/android/repository/{}".format(filename)
        engine.run(["mkdir", "-p", self.name()])
        filepath = os.path.join(self.name(), filename)
        engine.run(["curl", "-fsSLo", filepath, url])
        sha256file = os.path.join(cachedir(), "SDK", "SHA256")
        engine.echo_to_file(
            "{}  {}".format(android_cmdlinetools_sha256sum(), filepath), sha256file
        )
        engine.run(["shasum", "--check", sha256file])
        engine.run(["rm", sha256file])
        engine.run(["unzip", filename], cwd=self.name())
        engine.run(["rm", filepath])
        # See https://stackoverflow.com/a/61176718 to understand why
        # we need to reorganize the directories like this:
        engine.run(
            ["mv", "cmdline-tools", android_cmdlinetools_version()], cwd=self.name()
        )
        engine.run(["mkdir", "cmdline-tools"], cwd=self.name())
        engine.run(
            ["mv", android_cmdlinetools_version(), "cmdline-tools"], cwd=self.name()
        )
        engine.run(
            sdkmanager_install_cmd(
                os.path.join(
                    self.name(),
                    "cmdline-tools",
                    android_cmdlinetools_version(),
                    "bin",
                ),
            ),
            inputbytes=b"Y\n",  # automatically accept license
        )


class OONIProbePrivate(BaseTarget):
    """OONIProbePrivate creates ${cachedir}/github.com/ooni/probe-private."""

    # We use this private repository to copy the psiphon configuration
    # file to embed into the ooniprobe binaries

    def __init__(self) -> None:
        name = os.path.join(cachedir(), "github.com", "ooni", "probe-private")
        super().__init__(name, [])

    def copyfiles(self, engine: Engine, options: Options) -> None:
        """copyfiles copies psiphon config to the repository."""
        if options.disable_embedding_psiphon_config():
            log("./make: copy psiphon config: disabled by command line flags")
            return
        engine.run(
            [
                "cp",
                os.path.join(self.name(), "psiphon-config.json.age"),
                os.path.join("internal", "engine"),
            ]
        )
        engine.run(
            [
                "cp",
                os.path.join(self.name(), "psiphon-config.key"),
                os.path.join("internal", "engine"),
            ]
        )

    def build(self, engine: Engine, options: Options) -> None:
        if os.path.isdir(self.name()) and not options.dry_run():
            log("\n./make: {}: already built".format(self.name()))
            return
        if options.disable_embedding_psiphon_config():
            log("\n./make: {}: disabled by command line flags".format(self.name()))
            return
        log("\n./make: building {}...".format(self.name()))
        engine.require("git", "cp")
        engine.run(
            [
                "git",
                "clone",
                "git@github.com:ooni/probe-private",
                self.name(),
            ]
        )


class OONIMKAllAAR(BaseTarget):
    """OONIMKAllAAR creates ./MOBILE/android/oonimkall.aar."""

    def __init__(self) -> None:
        name = os.path.join(".", "MOBILE", "android", "oonimkall.aar")
        self._ooprivate = OONIProbePrivate()
        self._oonigo = SDKOONIGo()
        self._android = SDKAndroid()
        super().__init__(name, [self._ooprivate, self._oonigo, self._android])

    def aarfile(self) -> str:
        """aarfile returns the aar file path"""
        return self.name()

    def srcfile(self) -> str:
        """srcfile returns the path to the jar file containing sources."""
        return os.path.join(".", "MOBILE", "android", "oonimkall-sources.jar")

    def build(self, engine: Engine, options: Options) -> None:
        if os.path.isfile(self.name()) and not options.dry_run():
            log("\n./make: {}: already built".format(self.name()))
            return
        self.build_child_targets(engine, options)
        log("\n./make: building {}...".format(self.name()))
        self._ooprivate.copyfiles(engine, options)
        engine.require("sh", "javac")
        self._go_get_gomobile(engine, options, self._oonigo)
        self._gomobile_init(engine, self._oonigo, self._android)
        self._gomobile_bind(engine, options, self._oonigo, self._android)

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
        cmdline.append("go")
        cmdline.append("get")
        cmdline.append("-u")
        if options.verbose():
            cmdline.append("-v")
        if options.debugging():
            cmdline.append("-x")
        cmdline.append("golang.org/x/mobile/cmd/gomobile@latest")
        with Environ(engine, "GOPATH", gopath()):
            with AugmentedPath(engine, oonigo.binpath()):
                engine.require("go")
                engine.run(cmdline)

    def _gomobile_init(
        self,
        engine: Engine,
        oonigo: SDKOONIGo,
        android: SDKAndroid,
    ) -> None:
        cmdline: List[str] = []
        cmdline.append("gomobile")
        cmdline.append("init")
        with Environ(engine, "ANDROID_HOME", android.home()):
            with Environ(engine, "ANDROID_NDK_HOME", android.ndk_home()):
                with AugmentedPath(engine, oonigo.binpath()):
                    with AugmentedPath(engine, os.path.join(gopath(), "bin")):
                        engine.require("gomobile", "go")
                        engine.run(cmdline)

    def _gomobile_bind(
        self,
        engine: Engine,
        options: Options,
        oonigo: SDKOONIGo,
        android: SDKAndroid,
    ) -> None:
        cmdline: List[str] = []
        cmdline.append("gomobile")
        cmdline.append("bind")
        if options.verbose():
            cmdline.append("-v")
        if options.debugging():
            cmdline.append("-x")
        cmdline.append("-target")
        cmdline.append("android")
        cmdline.append("-o")
        cmdline.append(self.name())
        if not options.disable_embedding_psiphon_config():
            cmdline.append("-tags")
            cmdline.append("ooni_psiphon_config")
        cmdline.append("-ldflags")
        cmdline.append("-s -w")
        cmdline.append("./pkg/oonimkall")
        with Environ(engine, "ANDROID_HOME", android.home()):
            with Environ(engine, "ANDROID_NDK_HOME", android.ndk_home()):
                with AugmentedPath(engine, oonigo.binpath()):
                    with AugmentedPath(engine, os.path.join(gopath(), "bin")):
                        engine.require("gomobile", "go")
                        engine.run(cmdline)


def sign(engine: Engine, filepath: str) -> str:
    """sign signs the given filepath using pgp and returns
    the filepath of the signature file."""
    engine.require("gpg")
    user = "simone@openobservatory.org"
    engine.run(["gpg", "-abu", user, filepath])
    return filepath + ".asc"


class BundleJAR(BaseTarget):
    """BundleJAR creates ./MOBILE/android/bundle.jar."""

    # We upload the bundle.jar file to maven central to bless
    # a new release of the OONI libraries for Android.

    def __init__(self) -> None:
        name = os.path.join(".", "MOBILE", "android", "bundle.jar")
        self._oonimkall = OONIMKAllAAR()
        super().__init__(name, [self._oonimkall])

    def build(self, engine: Engine, options: Options) -> None:
        """build implements Target.build"""
        if os.path.isfile(self.name()) and not options.dry_run():
            log("\n./make: {}: already built".format(self.name()))
            return
        self.build_child_targets(engine, options)
        log("\n./make: building {}...".format(self.name()))
        engine.require("cp", "gpg", "jar")
        version = datetime.datetime.now().strftime("%Y.%m.%d-%H%M%S")
        engine.run(
            [
                "cp",
                self._oonimkall.aarfile(),
                os.path.join("MOBILE", "android", "oonimkall-{}.aar".format(version)),
            ]
        )
        engine.run(
            [
                "cp",
                self._oonimkall.srcfile(),
                os.path.join(
                    "MOBILE", "android", "oonimkall-{}-sources.jar".format(version)
                ),
            ]
        )
        engine.cat_sed_redirect(
            [
                ("@VERSION@", version),
            ],
            os.path.join("MOBILE", "template.pom"),
            os.path.join("MOBILE", "android", "oonimkall-{}.pom".format(version)),
        )
        names = (
            "oonimkall-{}.aar".format(version),
            "oonimkall-{}-sources.jar".format(version),
            "oonimkall-{}.pom".format(version),
        )
        allnames: List[str] = []
        with WorkingDir(os.path.join(".", "MOBILE", "android")):
            for name in names:
                allnames.append(name)
                allnames.append(sign(engine, name))
        engine.run(
            [
                "jar",
                "-cf",
                "bundle.jar",
                *allnames,
            ],
            cwd=os.path.join(".", "MOBILE", "android"),
        )


class Phony(BaseTarget):
    """Phony is a phony target that executes one or more other targets."""

    def __init__(self, name: str, depends: List[Target]):
        super().__init__(name, depends)

    def build(self, engine: Engine, options: Options) -> None:
        """build implements Target.build"""
        self.build_child_targets(engine, options)


# Android is the top-level "android" target
ANDROID = Phony("android", [BundleJAR()])


class OONIMKAllFramework(BaseTarget):
    """OONIMKAllFramework creates ./MOBILE/ios/oonimkall.framework."""

    def __init__(self) -> None:
        name = os.path.join(".", "MOBILE", "ios", "oonimkall.framework")
        self._ooprivate = OONIProbePrivate()
        self._gogo = SDKGolangGo()
        super().__init__(name, [self._ooprivate, self._gogo])

    def build(self, engine: Engine, options: Options) -> None:
        """build implements Target.build."""
        if os.path.isfile(self.name()) and not options.dry_run():
            log("\n./make: {}: already built".format(self.name()))
            return
        self.build_child_targets(engine, options)
        log("\n./make: building {}...".format(self.name()))
        self._ooprivate.copyfiles(engine, options)
        self._go_get_gomobile(engine, options, self._gogo)
        self._gomobile_init(engine, self._gogo)
        self._gomobile_bind(engine, options, self._gogo)

    def _go_get_gomobile(
        self,
        engine: Engine,
        options: Options,
        gogo: SDKGolangGo,
    ) -> None:
        # TODO(bassosimone): find a way to run this command without
        # adding extra dependencies to go.mod and go.sum.
        cmdline: List[str] = []
        cmdline.append("go")
        cmdline.append("get")
        cmdline.append("-u")
        if options.verbose():
            cmdline.append("-v")
        if options.debugging():
            cmdline.append("-x")
        cmdline.append("golang.org/x/mobile/cmd/gomobile@latest")
        with AugmentedPath(engine, gogo.binpath()):
            with Environ(engine, "GOPATH", gopath()):
                engine.require("go")
                engine.run(cmdline)

    def _gomobile_init(
        self,
        engine: Engine,
        gogo: SDKGolangGo,
    ) -> None:
        cmdline: List[str] = []
        cmdline.append("gomobile")
        cmdline.append("init")
        with AugmentedPath(engine, os.path.join(gopath(), "bin")):
            with AugmentedPath(engine, gogo.binpath()):
                engine.require("gomobile", "go")
                engine.run(cmdline)

    def _gomobile_bind(
        self,
        engine: Engine,
        options: Options,
        gogo: SDKGolangGo,
    ) -> None:
        cmdline: List[str] = []
        cmdline.append("gomobile")
        cmdline.append("bind")
        if options.verbose():
            cmdline.append("-v")
        if options.debugging():
            cmdline.append("-x")
        cmdline.append("-target")
        cmdline.append("ios")
        cmdline.append("-o")
        cmdline.append(self.name())
        if not options.disable_embedding_psiphon_config():
            cmdline.append("-tags")
            cmdline.append("ooni_psiphon_config")
        cmdline.append("-ldflags")
        cmdline.append("-s -w")
        cmdline.append("./pkg/oonimkall")
        with AugmentedPath(engine, os.path.join(gopath(), "bin")):
            with AugmentedPath(engine, gogo.binpath()):
                engine.require("gomobile", "go")
                engine.run(cmdline)


class OONIMKAllFrameworkZip(BaseTarget):
    """OONIMKAllFrameworkZip creates ./MOBILE/ios/oonimkall.framework.zip."""

    def __init__(self) -> None:
        name = os.path.join(".", "MOBILE", "ios", "oonimkall.framework.zip")
        ooframework = OONIMKAllFramework()
        super().__init__(name, [ooframework])

    def build(self, engine: Engine, options: Options) -> None:
        """build implements Target.build"""
        if os.path.isfile(self.name()) and not options.dry_run():
            log("\n./make: {}: already built".format(self.name()))
            return
        engine.require("zip", "rm")
        self.build_child_targets(engine, options)
        log("\n./make: building {}...".format(self.name()))
        engine.run(
            [
                "rm",
                "-rf",
                "oonimkall.framework.zip",
            ],
            cwd=os.path.join(".", "MOBILE", "ios"),
        )
        engine.run(
            [
                "zip",
                "-yr",
                "oonimkall.framework.zip",
                "oonimkall.framework",
            ],
            cwd=os.path.join(".", "MOBILE", "ios"),
        )


class OONIMKAllPodspec(BaseTarget):
    """OONIMKAllPodspec creates ./MOBILE/ios/oonimkall.podspec."""

    def __init__(self) -> None:
        name = os.path.join(".", "MOBILE", "ios", "oonimkall.podspec")
        super().__init__(name, [])

    def build(self, engine: Engine, options: Options) -> None:
        """build implements Target.build"""
        if os.path.isfile(self.name()) and not options.dry_run():
            log("./make: {}: already built".format(self.name()))
            return
        engine.require("cat", "sed")
        release = engine.backticks(["git", "describe", "--tags"])
        version = datetime.datetime.now().strftime("%Y.%m.%d-%H%M%S")
        engine.cat_sed_redirect(
            [("@VERSION@", version), ("@RELEASE@", release)],
            os.path.join(".", "MOBILE", "template.podspec"),
            self.name(),
        )


# IOS is the top-level "ios" target.
IOS = Phony("ios", [OONIMKAllFrameworkZip(), OONIMKAllPodspec()])


class MiniOONIDarwinOrWindows(BaseTarget):
    def __init__(self, goos: str, goarch: str):
        self._ext = ".exe" if goos == "windows" else ""
        self._os = goos
        self._arch = goarch
        name = os.path.join(".", "CLI", goos, goarch, "miniooni" + self._ext)
        self._ooprivate = OONIProbePrivate()
        self._gogo = SDKGolangGo()
        super().__init__(name, [self._ooprivate, self._gogo])

    def build(self, engine: Engine, options: Options) -> None:
        """build implements Target.build"""
        if os.path.isfile(self.name()) and not options.dry_run():
            log("\n./make: {}: already built".format(self.name()))
            return
        self.build_child_targets(engine, options)
        log("\n./make: building {}...".format(self.name()))
        self._ooprivate.copyfiles(engine, options)
        cmdline = [
            "go",
            "build",
            "-o",
            self.name(),
            "-ldflags=-s -w",
        ]
        if options.debugging():
            cmdline.append("-x")
        if options.verbose():
            cmdline.append("-v")
        if not options.disable_embedding_psiphon_config():
            cmdline.append("-tags=ooni_psiphon_config")
        cmdline.append("./internal/cmd/miniooni")
        with Environ(engine, "GOOS", self._os):
            with Environ(engine, "GOARCH", self._arch):
                with Environ(engine, "CGO_ENABLED", "0"):
                    with AugmentedPath(engine, self._gogo.binpath()):
                        engine.require("go")
                        engine.run(cmdline)


class MiniOONILinux(BaseTarget):
    def __init__(self, goarch: str):
        self._arch = goarch
        name = os.path.join(".", "CLI", "linux", goarch, "miniooni")
        self._ooprivate = OONIProbePrivate()
        self._gogo = SDKGolangGo()
        super().__init__(name, [self._ooprivate, self._gogo])

    def build(self, engine: Engine, options: Options) -> None:
        """build implements Target.build"""
        if os.path.isfile(self.name()) and not options.dry_run():
            log("\n./make: {}: already built".format(self.name()))
            return
        self.build_child_targets(engine, options)
        log("\n./make: building {}...".format(self.name()))
        self._ooprivate.copyfiles(engine, options)
        if self._arch == "arm":
            with Environ(engine, "GOARM", "7"):
                self._build(engine, options, self._gogo)
        else:
            self._build(engine, options, self._gogo)

    def _build(self, engine: Engine, options: Options, gogo: SDKGolangGo) -> None:
        cmdline = [
            "go",
            "build",
            "-o",
            os.path.join("CLI", "linux", self._arch, "miniooni"),
            "-ldflags=-s -w -extldflags -static",
        ]
        if options.debugging():
            cmdline.append("-x")
        if options.verbose():
            cmdline.append("-v")
        tags = "-tags=netgo"
        if not options.disable_embedding_psiphon_config():
            tags += ",ooni_psiphon_config"
        cmdline.append(tags)
        cmdline.append("./internal/cmd/miniooni")
        with Environ(engine, "GOOS", "linux"):
            with Environ(engine, "GOARCH", self._arch):
                with Environ(engine, "CGO_ENABLED", "0"):
                    with AugmentedPath(engine, gogo.binpath()):
                        engine.require("go")
                        engine.run(cmdline)


# MINIOONI is the top-level "miniooni" target.
MINIOONI = Phony(
    "miniooni",
    [
        MiniOONIDarwinOrWindows("darwin", "amd64"),
        MiniOONIDarwinOrWindows("darwin", "arm64"),
        MiniOONILinux("386"),
        MiniOONILinux("amd64"),
        MiniOONILinux("arm"),
        MiniOONILinux("arm64"),
        MiniOONIDarwinOrWindows("windows", "386"),
        MiniOONIDarwinOrWindows("windows", "amd64"),
    ],
)


class OONIProbeLinux(BaseTarget):
    """OONIProbeLinux builds ooniprobe for Linux."""

    # TODO(bassosimone): this works out of the box on macOS and
    # requires qemu-user-static on Fedora/Debian. I'm not sure what
    # is the right (set of) command(s) I should be checking for.

    def __init__(self, goarch: str):
        self._arch = goarch
        name = os.path.join(".", "CLI", "linux", goarch, "ooniprobe")
        self._ooprivate = OONIProbePrivate()
        super().__init__(name, [self._ooprivate])

    def build(self, engine: Engine, options: Options) -> None:
        """build implements Target.build."""
        if os.path.isfile(self.name()) and not options.dry_run():
            log("\n./make: {}: already built".format(self.name()))
            return
        self.build_child_targets(engine, options)
        log("\n./make: building {}...".format(self.name()))
        self._ooprivate.copyfiles(engine, options)
        engine.require("docker")
        # make sure we have the latest version of the container image
        engine.run(
            [
                "docker",
                "pull",
                "--platform",
                "linux/{}".format(self._arch),
                "golang:{}-alpine".format(goversion()),
            ]
        )
        # then run the build inside the container
        cmdline = [
            "docker",
            "run",
            "--platform",
            "linux/{}".format(self._arch),
            "-e",
            "GOARCH={}".format(self._arch),
            "-v",
            "{}:/ooni".format(os.getcwd()),
            "-w",
            "/ooni",
            "golang:{}-alpine".format(goversion()),
            os.path.join(".", "CLI", "linux", "build"),
        ]
        if options.debugging():
            cmdline.append("-x")
        if options.verbose():
            cmdline.append("-v")
        if not options.disable_embedding_psiphon_config():
            cmdline.append("-tags=ooni_psiphon_config,netgo")
        else:
            cmdline.append("-tags=netgo")
        engine.run(cmdline)


class OONIProbeWindows(BaseTarget):
    """OONIProbeWindows builds ooniprobe for Windows."""

    def __init__(self, goarch: str):
        self._arch = goarch
        name = os.path.join(".", "CLI", "windows", goarch, "ooniprobe.exe")
        self._ooprivate = OONIProbePrivate()
        self._gogo = SDKGolangGo()
        super().__init__(name, [self._ooprivate, self._gogo])

    def _gcc(self) -> str:
        if self._arch == "amd64":
            return "x86_64-w64-mingw32-gcc"
        if self._arch == "386":
            return "i686-w64-mingw32-gcc"
        raise NotImplementedError

    def build(self, engine: Engine, options: Options) -> None:
        """build implements Target.build"""
        if os.path.isfile(self.name()) and not options.dry_run():
            log("\n./make: {}: already built".format(self.name()))
            return
        self.build_child_targets(engine, options)
        log("\n./make: building {}...".format(self.name()))
        self._ooprivate.copyfiles(engine, options)
        cmdline = [
            "go",
            "build",
            "-o",
            self.name(),
            "-ldflags=-s -w",
        ]
        if options.debugging():
            cmdline.append("-x")
        if options.verbose():
            cmdline.append("-v")
        if not options.disable_embedding_psiphon_config():
            cmdline.append("-tags=ooni_psiphon_config")
        cmdline.append("./cmd/ooniprobe")
        with Environ(engine, "GOOS", "windows"):
            with Environ(engine, "GOARCH", self._arch):
                with Environ(engine, "CGO_ENABLED", "1"):
                    with Environ(engine, "CC", self._gcc()):
                        with AugmentedPath(engine, self._gogo.binpath()):
                            engine.require(self._gcc(), "go")
                            engine.run(cmdline)


class OONIProbeDarwin(BaseTarget):
    """OONIProbeDarwin builds ooniprobe for macOS."""

    def __init__(self, goarch: str):
        self._arch = goarch
        name = os.path.join(".", "CLI", "darwin", goarch, "ooniprobe")
        self._ooprivate = OONIProbePrivate()
        self._gogo = SDKGolangGo()
        super().__init__(name, [self._ooprivate, self._gogo])

    def build(self, engine: Engine, options: Options) -> None:
        """build implements Target.build"""
        if os.path.isfile(self.name()) and not options.dry_run():
            log("\n./make: {}: already built".format(self.name()))
            return
        self.build_child_targets(engine, options)
        log("\n./make: building {}...".format(self.name()))
        self._ooprivate.copyfiles(engine, options)
        cmdline = [
            "go",
            "build",
            "-o",
            self.name(),
            "-ldflags=-s -w",
        ]
        if options.debugging():
            cmdline.append("-x")
        if options.verbose():
            cmdline.append("-v")
        if not options.disable_embedding_psiphon_config():
            cmdline.append("-tags=ooni_psiphon_config")
        cmdline.append("./cmd/ooniprobe")
        with Environ(engine, "GOOS", "darwin"):
            with Environ(engine, "GOARCH", self._arch):
                with Environ(engine, "CGO_ENABLED", "1"):
                    with AugmentedPath(engine, self._gogo.binpath()):
                        engine.require("gcc", "go")
                        engine.run(cmdline)


class Debian(BaseTarget):
    """Debian makes a debian package for a given architecture."""

    def __init__(self, arch: str):
        name = "debian_{}".format(arch)
        self._target = OONIProbeLinux(arch)
        self._arch = arch
        super().__init__(name, [self._target])

    def build(self, engine: Engine, options: Options) -> None:
        self.build_child_targets(engine, options)
        log("\n./make: building {}...".format(self.name()))
        engine.require("docker")
        # make sure we have the latest version of the container image
        engine.run(
            [
                "docker",
                "pull",
                "--platform",
                "linux/{}".format(self._arch),
                "debian:stable",
            ]
        )
        # then run the build inside the container
        cmdline: List[str] = []
        cmdline.append("docker")
        cmdline.append("run")
        cmdline.append("--platform")
        cmdline.append("linux/{}".format(self._arch))
        cmdline.append("-v")
        cmdline.append("{}:/ooni".format(os.getcwd()))
        cmdline.append("-w")
        cmdline.append("/ooni")
        cmdline.append("debian:stable")
        cmdline.append(os.path.join(".", "CLI", "linux", "debian"))
        if os.environ.get("GITHUB_ACTIONS", "") == "true":
            # When we're running inside a github action, figure out whether
            # we are building a tag or a commit. In the latter case, we will
            # append the run number to the version number.
            github_ref = os.environ.get("GITHUB_REF")
            if not github_ref:
                raise RuntimeError("missing GITHUB_REF")
            github_run_number = os.environ.get("GITHUB_RUN_NUMBER")
            if not github_run_number:
                raise RuntimeError("missing GITHUB_RUN_NUMBER")
            if not github_ref.startswith("/refs/tags/"):
                cmdline.append(github_run_number)
        engine.run(cmdline)


class Sign(BaseTarget):
    """Sign signs a specific target artefact."""

    def __init__(self, target: Target):
        self._target = target
        name = self._target.name() + ".asc"
        super().__init__(name, [self._target])

    def build(self, engine: Engine, options: Options) -> None:
        if os.path.isfile(self.name()) and not options.dry_run():
            log("\n./make: {}: already built".format(self.name()))
            return
        self.build_child_targets(engine, options)
        log("\n./make: building {}...".format(self.name()))
        sign(engine, self._target.name())


# OONIPROBE_RELEASE_DARWIN contains the release darwin targets
OONIPROBE_RELEASE_DARWIN = Phony(
    "ooniprobe_release_darwin",
    [
        Sign(OONIProbeDarwin("amd64")),
        Sign(OONIProbeDarwin("arm64")),
    ],
)

# OONIPROBE_RELEASE_LINUX contains the release linux targets
OONIPROBE_RELEASE_LINUX = Phony(
    "ooniprobe_release_linux",
    [
        Sign(OONIProbeLinux("amd64")),
        Sign(OONIProbeLinux("arm64")),
    ],
)

# OONIPROBE_RELEASE_WINDOWS contains the release windows targets
OONIPROBE_RELEASE_WINDOWS = Phony(
    "ooniprobe_release_windows",
    [
        Sign(OONIProbeWindows("amd64")),
        Sign(OONIProbeWindows("386")),
    ],
)

# DEBIAN is the top-level "debian" target.
DEBIAN = Phony(
    "debian",
    [
        Debian("arm64"),
        Debian("amd64"),
    ],
)

# TOP_TARGETS contains the top-level targets
TOP_TARGETS: List[Target] = [
    ANDROID,
    IOS,
    MINIOONI,
    OONIPROBE_RELEASE_DARWIN,
    OONIPROBE_RELEASE_LINUX,
    OONIPROBE_RELEASE_WINDOWS,
    DEBIAN,
]


def expand_targets(targets: List[Target]) -> Dict[str, Target]:
    """expand_targets creates a dictionary mapping every existing
    target name to its implementation."""
    out: Dict[str, Target] = {}
    for tgt in targets:
        out.update(expand_targets(tgt.deps()))
        out[tgt.name()] = tgt
    return out


def cleanup_cachedir_sdk_golang() -> None:
    """cleanup_cachedir_sdk_golang deletes a previous download of
    the golang sdk that occurred in the cachedir. Since 2021-05-09,
    we download the official golang sdk at $HOME/sdk like VSCode
    does, to avoid duplicating the download and to simplify the
    code that manages downloading the Go SDK."""
    prevdir = os.path.join(cachedir(), "SDK", "golang")
    if os.path.isdir(prevdir):
        log("./make: # cleaning up legacy directory {}...".format(prevdir))
        log("./make: rm -rf '{}'".format(prevdir))
        shutil.rmtree(prevdir)


# CLEANUPS contains all the cleanups
CLEANUPS: List[Callable[[], None]] = [
    cleanup_cachedir_sdk_golang,
]


def main() -> None:
    """main function"""
    alltargets = expand_targets(TOP_TARGETS)
    options = ConfigFromCLI.parse(list(alltargets.keys()), TOP_TARGETS)
    for cleanup in CLEANUPS:
        cleanup()
    engine = new_engine(options)
    # note that we check whether the target is known in parse()
    selected = alltargets[options.target()]
    selected.build(engine, options)


if __name__ == "__main__":
    main()

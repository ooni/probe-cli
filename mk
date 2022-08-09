#!/usr/bin/make -f

# Many rules in here break if run in parallel.
.NOTPARALLEL:

#quickhelp: Usage: ./mk [VARIABLE=VALUE ...] TARGET ...
.PHONY: usage
usage:
	@cat mk | grep '^#quickhelp:' | sed -e 's/^#quickhelp://' -e 's/^\ *//'

# Most targets are .PHONY because whether to rebuild is controlled
# by golang. We expose to the user all the .PHONY targets.
#quickhelp:
#quickhelp: The `./mk list-targets` command lists all available targets.
.PHONY: list-targets
list-targets:
	@cat mk | grep '^\.PHONY:' | sed -e 's/^\.PHONY://'

#quickhelp:
#quickhelp: The `./mk help` command provides detailed usage instructions. We
#quickhelp: recommend running `./mk help|less` to page its output.
.PHONY: help
help:
	@cat mk | grep -E '^#(quick)?help:' | sed -E -e 's/^#(quick)?help://' -e s'/^\ //'

#help:
#help: The following variables control the build. You can specify them
#help: on the command line as a key-value pairs (see usage above).

#help:
#help: * ANDROID_CLI_SHA256    : the SHA256 of the Android CLI tools file. We always
#help:                           download the Linux version, which seems to work
#help:                           also on macOS (thank you, Java! :pray:).
ANDROID_CLI_SHA256 = 5e7bf2dd563d34917d32f3c5920a85562a795c93

#help:
#help: * ANDROID_CLI_VERSION   : the version of the Android CLI tools.
ANDROID_CLI_VERSION = 8512546

#help:
#help: * ANDROID_INSTALL_EXTRA : contains the android tools we install in addition
#help:                           to the NDK in order to build oonimkall.aar.
ANDROID_INSTALL_EXTRA = 'build-tools;32.0.0' 'platforms;android-31'

#help:
#help: * ANDROID_NDK_VERSION   : Android NDK version.
ANDROID_NDK_VERSION = 23.1.7779620

#help:
#help: * GIT_CLONE_DIR         : directory where to clone repositories, by default
#help:                           set to `$HOME/.ooniprobe-build/src`.
GIT_CLONE_DIR = $(HOME)/.ooniprobe-build/src

# $(GIT_CLONE_DIR) is an internal target that creates $(GIT_CLONE_DIR).
$(GIT_CLONE_DIR):
	mkdir -p $(GIT_CLONE_DIR)

#help:
#help: * GOLANG_EXTRA_FLAGS    : extra flags passed to `go build ...`, empty by
#help:                           default. Useful to pass flags to `go`, e.g.:
#help:
#help:                               ./mk GOLANG_EXTRA_FLAGS="-x -v" ./CLI/miniooni
GOLANG_EXTRA_FLAGS =

#help:
#help: * OONI_PSIPHON_TAGS     : build tags for `go build -tags ...` that cause
#help:                           the build to embed a psiphon configuration file
#help:                           into the generated binaries. This build tag
#help:                           implies cloning the git@github.com:ooni/probe-private
#help:                           repository. If you do not have the permission to
#help:                           clone it, just clear this variable, e.g.:
#help:
#help:                               ./mk OONI_PSIPHON_TAGS="" ./CLI/miniooni
OONI_PSIPHON_TAGS = ooni_psiphon_config

#help:
#help: * OONI_ANDROID_HOME     : directory where the Android SDK is downloaded
#help:                           and installed. You can point this to an existing
#help:                           copy of the SDK as long as (1) you have the
#help:                           right version of the command line tools, and
#help:                           (2) it's okay for us to install packages.
OONI_ANDROID_HOME = $(HOME)/.ooniprobe-build/sdk/android

#quickhelp:
#quickhelp: The `./mk show-config` command shows the current value of the
#quickhelp: variables controlling the build.
.PHONY: show-config
show-config:
	@echo "ANDROID_CLI_VERSION=$(ANDROID_CLI_VERSION)"
	@echo "ANDROID_CLI_SHA256=$(ANDROID_CLI_SHA256)"
	@echo "ANDROID_INSTALL_EXTRA=$(ANDROID_INSTALL_EXTRA)"
	@echo "ANDROID_NDK_VERSION=$(ANDROID_NDK_VERSION)"
	@echo "GIT_CLONE_DIR=$(GIT_CLONE_DIR)"
	@echo "GOLANG_EXTRA_FLAGS=$(GOLANG_EXTRA_FLAGS)"
	@echo "OONI_PSIPHON_TAGS=$(OONI_PSIPHON_TAGS)"
	@echo "OONI_ANDROID_HOME=$(OONI_ANDROID_HOME)"

# Cross-compiling miniooni from any system with Go installed is
# very easy, because it does not use any C code.
#help:
#help: The `./mk ./CLI/miniooni` command builds the miniooni experimental
#help: command line client for all the supported GOOS/GOARCH.
#help:
#help: You can also build the following subtargets:
.PHONY: ./CLI/miniooni
./CLI/miniooni: \
	./CLI/miniooni-darwin-amd64 \
	./CLI/miniooni-darwin-arm64 \
	./CLI/miniooni-linux-386 \
	./CLI/miniooni-linux-amd64 \
	./CLI/miniooni-linux-armv6 \
	./CLI/miniooni-linux-armv7 \
	./CLI/miniooni-linux-arm64 \
	./CLI/miniooni-windows-386.exe \
	./CLI/miniooni-windows-amd64.exe

# All the miniooni targets build with CGO_ENABLED=0 such that the build
# succeeds when the GOOS/GOARCH is such that we aren't crosscompiling
# (e.g., targeting darwin/amd64 on darwin/amd64) _and_ there's no C compiler
# installed on the system. We can afford that since miniooni is pure Go.
#help:
#help: * `./mk ./CLI/miniooni-darwin-amd64`: darwin/amd64
.PHONY:   ./CLI/miniooni-darwin-amd64
./CLI/miniooni-darwin-amd64: search/for/go maybe/copypsiphon
	./CLI/go-build-cross darwin amd64 ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/miniooni-darwin-arm64`: darwin/arm64
.PHONY:   ./CLI/miniooni-darwin-arm64
./CLI/miniooni-darwin-arm64: search/for/go maybe/copypsiphon
	./CLI/go-build-cross darwin arm64 ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/miniooni-linux-386`: linux/386
.PHONY:   ./CLI/miniooni-linux-386
./CLI/miniooni-linux-386: search/for/go maybe/copypsiphon
	./CLI/go-build-cross linux 386 ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/miniooni-linux-amd64`: linux/amd64
.PHONY:   ./CLI/miniooni-linux-amd64
./CLI/miniooni-linux-amd64: search/for/go maybe/copypsiphon
	./CLI/go-build-cross linux amd64 ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/miniooni-linux-armv6`: linux/armv6
.PHONY:   ./CLI/miniooni-linux-armv6
./CLI/miniooni-linux-armv6: search/for/go maybe/copypsiphon
	./CLI/go-build-cross linux armv6 ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/miniooni-linux-armv7`: linux/armv7
.PHONY:   ./CLI/miniooni-linux-armv7
./CLI/miniooni-linux-armv7: search/for/go maybe/copypsiphon
	./CLI/go-build-cross linux armv7 ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/miniooni-linux-arm64`: linux/arm64
.PHONY:   ./CLI/miniooni-linux-arm64
./CLI/miniooni-linux-arm64: search/for/go maybe/copypsiphon
	./CLI/go-build-cross linux arm64 ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/miniooni-windows-386.exe`: windows/386
.PHONY:   ./CLI/miniooni-windows-386.exe
./CLI/miniooni-windows-386.exe: search/for/go maybe/copypsiphon
	./CLI/go-build-cross windows 386 ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/miniooni-windows-amd64.exe`: windows/amd64
.PHONY:   ./CLI/miniooni-windows-amd64.exe
./CLI/miniooni-windows-amd64.exe: search/for/go maybe/copypsiphon
	./CLI/go-build-cross windows amd64 ./internal/cmd/miniooni

#help:
#help: The `./mk ./CLI/ooniprobe-darwin` command builds the ooniprobe official
#help: command line client for darwin/amd64 and darwin/arm64.
#help:
#help: You can also build the following subtargets:
.PHONY: ./CLI/ooniprobe-darwin
./CLI/ooniprobe-darwin: ./CLI/ooniprobe-darwin-amd64 ./CLI/ooniprobe-darwin-arm64

# We force CGO_ENABLED=1 because in principle we may be cross compiling. In
# reality it's hard to see a macOS/darwin build not made on macOS.
#help:
#help: * `./mk ./CLI/ooniprobe-darwin-amd64`: darwin/amd64
.PHONY:     ./CLI/ooniprobe-darwin-amd64
./CLI/ooniprobe-darwin-amd64: search/for/go maybe/copypsiphon
	./CLI/go-build-darwin amd64 ./cmd/ooniprobe

#help:
#help: * `./mk ./CLI/ooniprobe-darwin-arm64`: darwin/arm64
.PHONY:     ./CLI/ooniprobe-darwin-arm64
./CLI/ooniprobe-darwin-arm64: search/for/go maybe/copypsiphon
	./CLI/go-build-darwin arm64 ./cmd/ooniprobe

#help:
#help: The `./mk ./CLI/ooniprobe-linux` command builds the ooniprobe official command
#help: line client for amd64, arm64, etc.
#help:
#help: You can also build the following subtargets:
.PHONY: ./CLI/ooniprobe-linux
./CLI/ooniprobe-linux: \
	./CLI/ooniprobe-linux-386 \
	./CLI/ooniprobe-linux-amd64 \
	./CLI/ooniprobe-linux-armv7 \
	./CLI/ooniprobe-linux-arm64

# Linux builds use Alpine and Docker so we are sure that we are statically
# linking to musl libc, thus making our binaries extremely portable.
#help:
#help: * `./mk ./CLI/ooniprobe-linux-386`: linux/386
.PHONY:     ./CLI/ooniprobe-linux-386
./CLI/ooniprobe-linux-386: search/for/docker maybe/copypsiphon
	./CLI/go-build-linux-static 386 ./cmd/ooniprobe

#help:
#help: * `./mk ./CLI/ooniprobe-linux-amd64`: linux/amd64
.PHONY:     ./CLI/ooniprobe-linux-amd64
./CLI/ooniprobe-linux-amd64: search/for/docker maybe/copypsiphon
	./CLI/go-build-linux-static amd64 ./cmd/ooniprobe

#help:
#help: * `./mk ./CLI/ooniprobe-linux-armv7`: linux/arm
.PHONY:     ./CLI/ooniprobe-linux-armv7
./CLI/ooniprobe-linux-armv7: search/for/docker maybe/copypsiphon
	./CLI/go-build-linux-static armv7 ./cmd/ooniprobe

#help:
#help: * `./mk ./CLI/ooniprobe-linux-arm64`: linux/arm64
.PHONY:     ./CLI/ooniprobe-linux-arm64
./CLI/ooniprobe-linux-arm64: search/for/docker maybe/copypsiphon
	./CLI/go-build-linux-static arm64 ./cmd/ooniprobe

#help:
#help: The `./mk ./CLI/ooniprobe-windows` command builds the ooniprobe official
#help: command line client for windows/386 and windows/amd64.
#help:
#help: You can also build the following subtargets:
.PHONY: ./CLI/ooniprobe-windows
./CLI/ooniprobe-windows: \
	./CLI/ooniprobe-windows-386.exe \
	./CLI/ooniprobe-windows-amd64.exe

#help:
#help: * `./mk ./CLI/ooniprobe-windows-386.exe`: windows/386
.PHONY:     ./CLI/ooniprobe-windows-386.exe
./CLI/ooniprobe-windows-386.exe: search/for/go search/for/mingw-w64 maybe/copypsiphon
	./CLI/go-build-windows 386 ./cmd/ooniprobe

#help:
#help: * `./mk ./CLI/ooniprobe-windows-amd64.exe`: windows/amd64
.PHONY:     ./CLI/ooniprobe-windows-amd64.exe
./CLI/ooniprobe-windows-amd64.exe: search/for/go search/for/mingw-w64 maybe/copypsiphon
	./CLI/go-build-windows amd64 ./cmd/ooniprobe

#help:
#help: The `./mk ./MOBILE/android` command builds the oonimkall library for Android.
#help:
#help: You can also build the following subtargets:
.PHONY: ./MOBILE/android
./MOBILE/android: ./MOBILE/android/oonimkall.aar ./MOBILE/android/oonimkall.pom

#help:
#help: * `./mk ./MOBILE/android/oonimkall.pom`: the POM
.PHONY:   ./MOBILE/android/oonimkall.pom
./MOBILE/android/oonimkall.pom:
	cat ./MOBILE/android/template.pom | sed -e "s/@VERSION@/$(OONIMKALL_V)/g" > ./MOBILE/android/oonimkall.pom

#help:
#help: * `./mk ./MOBILE/android/oonimkall.aar`: the AAR
.PHONY:   ./MOBILE/android/oonimkall.aar
./MOBILE/android/oonimkall.aar: android/sdk maybe/copypsiphon
	@echo "Android build disabled - TODO(https://github.com/ooni/probe/issues/2122)"
	@exit 1
	./MOBILE/gomobile android ./pkg/oonimkall

#help:
#help: The `./mk ./MOBILE/ios` command builds the oonimkall library for iOS.
#help:
#help: You can also build the following subtargets:
.PHONY: ./MOBILE/ios
./MOBILE/ios: ./MOBILE/ios/oonimkall.xcframework.zip ./MOBILE/ios/oonimkall.podspec

#help:
#help: * `./mk ./MOBILE/ios/oonimkall.xcframework.zip`: zip the xcframework
.PHONY:   ./MOBILE/ios/oonimkall.xcframework.zip
./MOBILE/ios/oonimkall.xcframework.zip: search/for/zip ./MOBILE/ios/oonimkall.xcframework
	cd ./MOBILE/ios && rm -rf oonimkall.xcframework.zip
	cd ./MOBILE/ios && zip -yr oonimkall.xcframework.zip oonimkall.xcframework

#help:
#help: * `./mk ./MOBILE/ios/xcframework`: the xcframework
.PHONY:     ./MOBILE/ios/oonimkall.xcframework
./MOBILE/ios/oonimkall.xcframework: search/for/go search/for/xcode maybe/copypsiphon
	./MOBILE/gomobile ios ./pkg/oonimkall

#help:
#help: * `./mk ./MOBILE/ios/oonimkall.podspec`: the podspec
.PHONY:   ./MOBILE/ios/oonimkall.podspec
./MOBILE/ios/oonimkall.podspec: ./MOBILE/ios/template.podspec
	cat $< | sed -e "s/@VERSION@/$(OONIMKALL_V)/g" -e "s/@RELEASE@/$(OONIMKALL_R)/g" > $@

# important: OONIMKALL_V and OONIMKALL_R MUST be expanded just once so we use `:=`
OONIMKALL_V := $(shell date -u +%Y.%m.%d-%H%M%S)
OONIMKALL_R := $(shell git describe --tags || echo '0.0.0-dev')

#help:
#help: The following commands check for the availability of dependencies:
# TODO(bassosimone): make checks more robust?

#help:
#help: * `./mk search/for/bash`: checks for bash
.PHONY: search/for/bash
search/for/bash:
	@printf "checking for bash... "
	@command -v bash || { echo "not found"; exit 1; }

#help:
#help: * `./mk search/for/curl`: checks for curl
.PHONY: search/for/curl
search/for/curl:
	@printf "checking for curl... "
	@command -v curl || { echo "not found"; exit 1; }

#help:
#help: * `./mk search/for/docker`: checks for docker
.PHONY: search/for/docker
search/for/docker:
	@printf "checking for docker... "
	@command -v docker || { echo "not found"; exit 1; }

#help:
#help: * `./mk search/for/git`: checks for git
.PHONY: search/for/git
search/for/git:
	@printf "checking for git... "
	@command -v git || { echo "not found"; exit 1; }

.PHONY: search/for/go
search/for/go:
	./CLI/check-go-version

#help:
#help: * `./mk search/for/java`: checks for java
.PHONY: search/for/java
search/for/java:
	@printf "checking for java... "
	@command -v java || { echo "not found"; exit 1; }

.PHONY: search/for/mingw-w64
search/for/mingw-w64:
	./CLI/check-mingw-w64-version

#help:
#help: * `./mk search/for/shasum`: checks for shasum
.PHONY: search/for/shasum
search/for/shasum:
	@printf "checking for shasum... "
	@command -v shasum || { echo "not found"; exit 1; }

.PHONY: search/for/xcode
search/for/xcode:
	./MOBILE/ios/check-xcode-version

#help:
#help: * `./mk search/for/unzip`: checks for unzip
.PHONY: search/for/unzip
search/for/unzip:
	@printf "checking for unzip... "
	@command -v unzip || { echo "not found"; exit 1; }

#help:
#help: * `./mk search/for/zip`: checks for zip
.PHONY: search/for/zip
search/for/zip:
	@printf "checking for zip... "
	@command -v zip || { echo "not found"; exit 1; }

#help:
#help: The `./mk maybe/copypsiphon` command checks whether we want
#help: to embed the Psiphon config file into the build. To this end,
#help: this command checks whether OONI_PSIPHON_TAGS is set. In
#help: such a case, this command checks whether the required files
#help: are already in place. If not, this command fetches them
#help: by cloning the github.com/ooni/probe-private repo.
#
# Note: we check for files being already there before attempting
# to clone _because_ we put files in there using secrets when
# running cloud builds. This saves us from including a token with
# `repo` scope as a build secret, which is a very broad scope.
#
# Cloning the private repository, instead, is the way in which
# local builds get access to the psiphon config files.
.PHONY: maybe/copypsiphon
maybe/copypsiphon: search/for/git
	@if test "$(OONI_PSIPHON_TAGS)" = "ooni_psiphon_config"; then \
		if test ! -f ./internal/engine/psiphon-config.json.age -a \
		        ! -f ./internal/engine/psiphon-config.key; then \
			echo "copying psiphon configuration file into ./internal/engine"; \
			$(MAKE) -f mk $(OONIPRIVATE) || exit 1; \
			cp $(OONIPRIVATE)/psiphon-config.key ./internal/engine || exit 1; \
			cp $(OONIPRIVATE)/psiphon-config.json.age ./internal/engine || exit 1; \
		fi; \
	fi

# OONIPRIVATE is the directory where we clone the private repository.
OONIPRIVATE = $(GIT_CLONE_DIR)/github.com/ooni/probe-private

# OONIPRIVATE_REPO is the private repository URL.
OONIPRIVATE_REPO = git@github.com:ooni/probe-private

# $(OONIPRIVATE) clones the private repository in $(GIT_CLONE_DIR)
$(OONIPRIVATE): search/for/git $(GIT_CLONE_DIR)
	rm -rf $(OONIPRIVATE)
	git clone $(OONIPRIVATE_REPO) $(OONIPRIVATE)

#help:
#help: The `./mk android/sdk` command ensures we are using the
#help: correct version of the Android sdk.
.PHONY: android/sdk
android/sdk: search/for/java
	rm -rf $(OONI_ANDROID_HOME)
	$(MAKE) -f mk android/sdk/download
	test -f $(__ANDROID_SDKMANAGER) || { echo "please run './mk android/sdk/download'"; exit 1; }
	echo "Yes" | $(__ANDROID_SDKMANAGER) --install $(ANDROID_INSTALL_EXTRA) 'ndk;$(ANDROID_NDK_VERSION)'

# __ANDROID_SKDMANAGER is the path to android's sdkmanager tool
__ANDROID_SDKMANAGER = $(OONI_ANDROID_HOME)/cmdline-tools/$(ANDROID_CLI_VERSION)/bin/sdkmanager

# See https://stackoverflow.com/a/61176718 to understand why
# we need to reorganize the directories like this:
#help:
#help: The `./mk android/sdk/download` unconditionally downloads the
#help: Android SDK at `$(OONI_ANDROID_HOME)`.
android/sdk/download: search/for/curl search/for/java search/for/shasum search/for/unzip
	curl -fsSLO https://dl.google.com/android/repository/$(__ANDROID_CLITOOLS_FILE)
	echo "$(ANDROID_CLI_SHA256)  $(__ANDROID_CLITOOLS_FILE)" > __SHA256
	shasum --check __SHA256
	rm -f __SHA256
	unzip $(__ANDROID_CLITOOLS_FILE)
	rm $(__ANDROID_CLITOOLS_FILE)
	mkdir -p $(OONI_ANDROID_HOME)/cmdline-tools
	mv cmdline-tools $(OONI_ANDROID_HOME)/cmdline-tools/$(ANDROID_CLI_VERSION)

# __ANDROID_CLITOOLS_FILE is the file name of the android cli tools zip
__ANDROID_CLITOOLS_FILE = commandlinetools-linux-$(ANDROID_CLI_VERSION)_latest.zip

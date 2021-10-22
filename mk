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
ANDROID_CLI_SHA256 = 7a00faadc0864f78edd8f4908a629a46d622375cbe2e5814e82934aebecdb622

#help:
#help: * ANDROID_CLI_VERSION   : the version of the Android CLI tools.
ANDROID_CLI_VERSION = 7302050

#help:
#help: * ANDROID_INSTALL_EXTRA : contains the android tools we install in addition
#help:                           to the NDK in order to build oonimkall.aar.
ANDROID_INSTALL_EXTRA = 'build-tools;29.0.3' 'platforms;android-30'

#help:
#help: * ANDROID_NDK_VERSION   : Android NDK version.
ANDROID_NDK_VERSION = 22.1.7171670

#help:
#help: * DEBIAN_TILDE_VERSION  : if non-empty, this should be "[0-9]+" and
#help:                           will be appended to the package version using
#help:                           a tilde, thus producing, e.g., "1.0~1234".
DEBIAN_TILDE_VERSION =

#help:
#help: * GIT_CLONE_DIR         : directory where to clone repositories, by default
#help:                           set to `$HOME/.ooniprobe-build/src`.
GIT_CLONE_DIR = $(HOME)/.ooniprobe-build/src

# $(GIT_CLONE_DIR) is an internal target that creates $(GIT_CLONE_DIR).
$(GIT_CLONE_DIR):
	mkdir -p $(GIT_CLONE_DIR)

#help:
#help: * GOLANG_DOCKER_GOCACHE : where to store golang's build cache to
#help:                           speed up subsequent Docker builds.
GOLANG_DOCKER_GOCACHE = $(HOME)/.ooniprobe-build/docker/gocache

#help:
#help: * GOLANG_DOCKER_GOPATH  : GOPATH directory used by builds running
#help:                           inside docker to significantly speed
#help:                           up subsequent Docker based builds.
GOLANG_DOCKER_GOPATH := $(HOME)/.ooniprobe-build/docker/gopath

#help:
#help: * GOLANG_EXTRA_FLAGS    : extra flags passed to `go build ...`, empty by
#help:                           default. Useful to pass flags to `go`, e.g.:
#help:
#help:                               ./mk GOLANG_EXTRA_FLAGS="-x -v" ./CLI/miniooni
GOLANG_EXTRA_FLAGS =

#help:
#help: * GOLANG_VERSION_NUMBER : the expected version number for golang.
GOLANG_VERSION_NUMBER = 1.17.2

#help:
#help: * GPG_USER              : allows overriding the default GPG user used
#help:                           to sign binary releases, e.g.:
#help:
#help:                               ./mk GPG_USER=john@doe.com ooniprobe/windows
GPG_USER = simone@openobservatory.org

#help:
#help: * MINGW_W64_VERSION     : the expected mingw-w64 version.
MINGW_W64_VERSION = 10.3.1

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

#help:
#help: * XCODE_VERSION         : the version of Xcode we expect.
XCODE_VERSION = 12.5

#quickhelp:
#quickhelp: The `./mk show-config` command shows the current value of the
#quickhelp: variables controlling the build.
.PHONY: show-config
show-config:
	@echo "ANDROID_CLI_VERSION=$(ANDROID_CLI_VERSION)"
	@echo "ANDROID_CLI_SHA256=$(ANDROID_CLI_SHA256)"
	@echo "ANDROID_INSTALL_EXTRA=$(ANDROID_INSTALL_EXTRA)"
	@echo "ANDROID_NDK_VERSION=$(ANDROID_NDK_VERSION)"
	@echo "DEBIAN_TILDE_VERSION=$(DEBIAN_TILDE_VERSION)"
	@echo "GIT_CLONE_DIR=$(GIT_CLONE_DIR)"
	@echo "GOLANG_DOCKER_GOCACHE=$(GOLANG_DOCKER_GOCACHE)"
	@echo "GOLANG_DOCKER_GOPATH=$(GOLANG_DOCKER_GOPATH)"
	@echo "GOLANG_EXTRA_FLAGS=$(GOLANG_EXTRA_FLAGS)"
	@echo "GOLANG_VERSION_NUMBER=$(GOLANG_VERSION_NUMBER)"
	@echo "GPG_USER=$(GPG_USER)"
	@echo "MINGW_W64_VERSION=$(MINGW_W64_VERSION)"
	@echo "OONI_PSIPHON_TAGS=$(OONI_PSIPHON_TAGS)"
	@echo "OONI_ANDROID_HOME=$(OONI_ANDROID_HOME)"
	@echo "XCODE_VERSION=$(XCODE_VERSION)"

# GOLANG_VERSION_STRING is the expected version string. If we
# run a golang binary that does not emit this version string
# when running `go version`, we stop the build.
GOLANG_VERSION_STRING = go$(GOLANG_VERSION_NUMBER)

# GOLANG_DOCKER_IMAGE is the golang docker image we use for
# building for Linux systems. It is an Alpine based container
# so that we can easily build static binaries.
GOLANG_DOCKER_IMAGE = golang:$(GOLANG_VERSION_NUMBER)-alpine

# Cross-compiling miniooni from any system with Go installed is
# very easy, because it does not use any C code.
#help:
#help: The `./mk ./CLI/miniooni` command builds the miniooni experimental
#help: command line client for all the supported GOOS/GOARCH.
#help:
#help: You can also build the following subtargets:
.PHONY: ./CLI/miniooni
./CLI/miniooni:                       \
	./CLI/darwin/amd64/miniooni       \
	./CLI/darwin/arm64/miniooni       \
	./CLI/linux/386/miniooni          \
	./CLI/linux/amd64/miniooni        \
	./CLI/linux/arm/miniooni          \
	./CLI/linux/arm64/miniooni        \
	./CLI/windows/386/miniooni.exe    \
	./CLI/windows/amd64/miniooni.exe

# All the miniooni targets build with CGO_ENABLED=0 such that the build
# succeeds when the GOOS/GOARCH is such that we aren't crosscompiling
# (e.g., targeting darwin/amd64 on darwin/amd64) _and_ there's no C compiler
# installed on the system. We can afford that since miniooni is pure Go.
#help:
#help: * `./mk ./CLI/darwin/amd64/miniooni`: darwin/amd64
.PHONY:   ./CLI/darwin/amd64/miniooni
./CLI/darwin/amd64/miniooni: search/for/go maybe/copypsiphon
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GOLANG_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/darwin/arm64/miniooni`: darwin/arm64
.PHONY:   ./CLI/darwin/arm64/miniooni
./CLI/darwin/arm64/miniooni: search/for/go maybe/copypsiphon
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GOLANG_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

# When building for Linux we use `-tags netgo` and `-extldflags -static` to produce
# a statically linked binary that completely bypasses libc.
#help:
#help: * `./mk ./CLI/linux/386/miniooni`: linux/386
.PHONY:   ./CLI/linux/386/miniooni
./CLI/linux/386/miniooni: search/for/go maybe/copypsiphon
	GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -tags="netgo,$(OONI_PSIPHON_TAGS)" -ldflags="-s -w -extldflags -static" $(GOLANG_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/linux/amd64/miniooni`: linux/amd64
.PHONY:   ./CLI/linux/amd64/miniooni
./CLI/linux/amd64/miniooni: search/for/go maybe/copypsiphon
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags="netgo,$(OONI_PSIPHON_TAGS)" -ldflags="-s -w -extldflags -static" $(GOLANG_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

# When building for GOARCH=arm, we always force GOARM=7 (i.e., armhf/armv7).
#help:
#help: * `./mk ./CLI/linux/arm/miniooni`: linux/arm
.PHONY:   ./CLI/linux/arm/miniooni
./CLI/linux/arm/miniooni: search/for/go maybe/copypsiphon
	GOOS=linux GOARCH=arm CGO_ENABLED=0 GOARM=7 go build -tags="netgo,$(OONI_PSIPHON_TAGS)" -ldflags="-s -w -extldflags -static" $(GOLANG_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/linux/arm64/miniooni`: linux/arm64
.PHONY:   ./CLI/linux/arm64/miniooni
./CLI/linux/arm64/miniooni: search/for/go maybe/copypsiphon
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags="netgo,$(OONI_PSIPHON_TAGS)" -ldflags="-s -w -extldflags -static" $(GOLANG_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/windows/386/miniooni.exe`: windows/386
.PHONY:   ./CLI/windows/386/miniooni.exe
./CLI/windows/386/miniooni.exe: search/for/go maybe/copypsiphon
	GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GOLANG_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/windows/amd64/miniooni.exe`: windows/amd64
.PHONY:   ./CLI/windows/amd64/miniooni.exe
./CLI/windows/amd64/miniooni.exe: search/for/go maybe/copypsiphon
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GOLANG_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: The `./mk ./CLI/ooniprobe/darwin` command builds the ooniprobe official
#help: command line client for darwin/amd64 and darwin/arm64. This process
#help: entails building ooniprobe and then GPG-signing the binaries.
#help:
#help: You can also build the following subtargets:
.PHONY: ./CLI/ooniprobe/darwin
./CLI/ooniprobe/darwin:                 \
	./ooniprobe_darwin_amd64.tar.gz.asc \
	./ooniprobe_darwin_arm64.tar.gz.asc

# ./ooniprobe_darwin_amd64.tar.gz.asc creates and signs the release tarball
.PHONY:   ./ooniprobe_darwin_amd64.tar.gz.asc
./ooniprobe_darwin_amd64.tar.gz.asc: ./CLI/darwin/amd64/ooniprobe
	rm -f ooniprobe_darwin_amd64.tar.gz ooniprobe_darwin_amd64.tar.gz.asc
	tar -cvzf ooniprobe_darwin_amd64.tar.gz -C ./CLI/darwin/amd64 ooniprobe
	gpg -abu $(GPG_USER) ooniprobe_darwin_amd64.tar.gz

# We force CGO_ENABLED=1 because in principle we may be cross compiling. In
# reality it's hard to see a macOS/darwin build not made on macOS.
#help:
#help: * `./mk ./CLI/darwin/amd64/ooniprobe`: darwin/amd64
.PHONY:     ./CLI/darwin/amd64/ooniprobe
./CLI/darwin/amd64/ooniprobe: search/for/go maybe/copypsiphon
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GOLANG_EXTRA_FLAGS) -o $@ ./cmd/ooniprobe

# ./ooniprobe_darwin_arm64.tar.gz.asc creates and signs the release tarball
.PHONY:   ./ooniprobe_darwin_arm64.tar.gz.asc
./ooniprobe_darwin_arm64.tar.gz.asc: ./CLI/darwin/arm64/ooniprobe
	rm -f ooniprobe_darwin_arm64.tar.gz ooniprobe_darwin_arm64.tar.gz.asc
	tar -cvzf ooniprobe_darwin_arm64.tar.gz -C ./CLI/darwin/arm64 ooniprobe
	gpg -abu $(GPG_USER) ooniprobe_darwin_arm64.tar.gz

#help:
#help: * `./mk ./CLI/darwin/arm64/ooniprobe`: darwin/arm64
.PHONY:     ./CLI/darwin/arm64/ooniprobe
./CLI/darwin/arm64/ooniprobe: search/for/go maybe/copypsiphon
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GOLANG_EXTRA_FLAGS) -o $@ ./cmd/ooniprobe

#help:
#help: The `./mk ./debian` command builds the ooniprobe CLI
#help: debian package for amd64 and arm64.
#help:
#help: You can also build the following subtargets:
.PHONY: ./debian
./debian:           \
	./debian/386    \
	./debian/amd64  \
	./debian/arm    \
	./debian/arm64

#help:
#help: * `./mk ./debian/386`: debian/386
.PHONY:   ./debian/386
# This extra .PHONY for linux/386 is to help printing targets 🤷.
.PHONY:     ./CLI/linux/386/ooniprobe
./debian/386: search/for/docker ./CLI/linux/386/ooniprobe
	docker pull debian:stable
	docker run -v $(shell pwd):/ooni -w /ooni debian:stable ./CLI/linux/pkgdebian 386 "$(DEBIAN_TILDE_VERSION)"

#help:
#help: * `./mk ./debian/amd64`: debian/amd64
.PHONY:   ./debian/amd64
# This extra .PHONY for linux/amd64 is to help printing targets 🤷.
.PHONY:     ./CLI/linux/amd64/ooniprobe
./debian/amd64: search/for/docker ./CLI/linux/amd64/ooniprobe
	docker pull debian:stable
	docker run -v $(shell pwd):/ooni -w /ooni debian:stable ./CLI/linux/pkgdebian amd64 "$(DEBIAN_TILDE_VERSION)"

# Note that we're building for armv7 here
#help:
#help: * `./mk ./debian/arm`: debian/arm
.PHONY:   ./debian/arm
# This extra .PHONY for linux/arm is to help printing targets 🤷.
.PHONY:     ./CLI/linux/arm/ooniprobe
./debian/arm: search/for/docker ./CLI/linux/arm/ooniprobe
	docker pull debian:stable
	docker run -v $(shell pwd):/ooni -w /ooni debian:stable ./CLI/linux/pkgdebian arm "$(DEBIAN_TILDE_VERSION)"

#help:
#help: * `./mk ./debian/arm64`: debian/arm64
.PHONY:   ./debian/arm64
# This extra .PHONY for linux/arm64 is to help printing targets 🤷.
.PHONY:     ./CLI/linux/arm64/ooniprobe
./debian/arm64: search/for/docker ./CLI/linux/arm64/ooniprobe
	docker pull debian:stable
	docker run -v $(shell pwd):/ooni -w /ooni debian:stable ./CLI/linux/pkgdebian arm64 "$(DEBIAN_TILDE_VERSION)"

#help:
#help: The `./mk ./CLI/ooniprobe/linux` command builds the ooniprobe official command
#help: line client for amd64 and arm64. This entails building and GPG signing.
#help:
#help: You can also build the following subtargets:
.PHONY: ./CLI/ooniprobe/linux
./CLI/ooniprobe/linux:                 \
	./ooniprobe_linux_386.tar.gz.asc   \
	./ooniprobe_linux_amd64.tar.gz.asc \
	./ooniprobe_linux_armv7.tar.gz.asc \
	./ooniprobe_linux_arm64.tar.gz.asc

# ./ooniprobe_linux_386.tar.gz.asc creates and signs the release tarball
.PHONY:   ./ooniprobe_linux_386.tar.gz.asc
./ooniprobe_linux_386.tar.gz.asc: ./CLI/linux/386/ooniprobe
	rm -f ooniprobe_linux_386.tar.gz ooniprobe_linux_386.tar.gz.asc
	tar -cvzf ooniprobe_linux_386.tar.gz -C ./CLI/linux/386 ooniprobe
	gpg -abu $(GPG_USER) ooniprobe_linux_386.tar.gz

# Linux builds use Alpine and Docker so we are sure that we are statically
# linking to musl libc, thus making our binaries extremely portable.
#help:
#help: * `./mk ./CLI/linux/386/ooniprobe`: linux/386
.PHONY:     ./CLI/linux/386/ooniprobe
./CLI/linux/386/ooniprobe: search/for/docker maybe/copypsiphon
	docker pull --platform linux/386 $(GOLANG_DOCKER_IMAGE)
	docker run --platform linux/386 -e GOPATH=/gopath -e GOARCH=386 -v $(GOLANG_DOCKER_GOCACHE)/386:/root/.cache/go-build -v $(GOLANG_DOCKER_GOPATH):/gopath -v $(shell pwd):/ooni -w /ooni $(GOLANG_DOCKER_IMAGE) ./CLI/linux/build -tags=netgo,$(OONI_PSIPHON_TAGS) $(GOLANG_EXTRA_FLAGS)

# ./ooniprobe_linux_amd64.tar.gz.asc creates and signs the release tarball
.PHONY:   ./ooniprobe_linux_amd64.tar.gz.asc
./ooniprobe_linux_amd64.tar.gz.asc: ./CLI/linux/amd64/ooniprobe
	rm -f ooniprobe_linux_amd64.tar.gz ooniprobe_linux_amd64.tar.gz.asc
	tar -cvzf ooniprobe_linux_amd64.tar.gz -C ./CLI/linux/amd64 ooniprobe
	gpg -abu $(GPG_USER) ooniprobe_linux_amd64.tar.gz

#help:
#help: * `./mk ./CLI/linux/amd64/ooniprobe`: linux/amd64
.PHONY:     ./CLI/linux/amd64/ooniprobe
./CLI/linux/amd64/ooniprobe: search/for/docker maybe/copypsiphon
	docker pull --platform linux/amd64 $(GOLANG_DOCKER_IMAGE)
	docker run --platform linux/amd64 -e GOPATH=/gopath -e GOARCH=amd64 -v $(GOLANG_DOCKER_GOCACHE)/amd64:/root/.cache/go-build -v $(GOLANG_DOCKER_GOPATH):/gopath -v $(shell pwd):/ooni -w /ooni $(GOLANG_DOCKER_IMAGE) ./CLI/linux/build -tags=netgo,$(OONI_PSIPHON_TAGS) $(GOLANG_EXTRA_FLAGS)

# ./ooniprobe_linux_armv7.tar.gz.asc creates and signs the release tarball
.PHONY:   ./ooniprobe_linux_armv7.tar.gz.asc
./ooniprobe_linux_armv7.tar.gz.asc: ./CLI/linux/arm/ooniprobe
	rm -f ooniprobe_linux_armv7.tar.gz ooniprobe_linux_armv7.tar.gz.asc
	tar -cvzf ooniprobe_linux_armv7.tar.gz -C ./CLI/linux/arm ooniprobe
	gpg -abu $(GPG_USER) ooniprobe_linux_armv7.tar.gz

# Note that we're building for armv7 here
#help:
#help: * `./mk ./CLI/linux/arm/ooniprobe`: linux/arm
.PHONY:     ./CLI/linux/arm/ooniprobe
./CLI/linux/arm/ooniprobe: search/for/docker maybe/copypsiphon
	docker pull --platform linux/arm/v7 $(GOLANG_DOCKER_IMAGE)
	docker run --platform linux/arm/v7 -e GOPATH=/gopath -e GOARCH=arm -e GOARM=7 -v $(GOLANG_DOCKER_GOCACHE)/arm:/root/.cache/go-build -v $(GOLANG_DOCKER_GOPATH):/gopath -v $(shell pwd):/ooni -w /ooni $(GOLANG_DOCKER_IMAGE) ./CLI/linux/build -tags=netgo,$(OONI_PSIPHON_TAGS) $(GOLANG_EXTRA_FLAGS)

# ./ooniprobe_linux_arm64.tar.gz.asc creates and signs the release tarball
.PHONY:   ./ooniprobe_linux_arm64.tar.gz.asc
./ooniprobe_linux_arm64.tar.gz.asc: ./CLI/linux/arm64/ooniprobe
	rm -f ooniprobe_linux_arm64.tar.gz ooniprobe_linux_arm64.tar.gz.asc
	tar -cvzf ooniprobe_linux_arm64.tar.gz -C ./CLI/linux/arm64 ooniprobe
	gpg -abu $(GPG_USER) ooniprobe_linux_arm64.tar.gz

#help:
#help: * `./mk ./CLI/linux/arm64/ooniprobe`: linux/arm64
.PHONY:     ./CLI/linux/arm64/ooniprobe
./CLI/linux/arm64/ooniprobe: search/for/docker maybe/copypsiphon
	docker pull --platform linux/arm64 $(GOLANG_DOCKER_IMAGE)
	docker run --platform linux/arm64 -e GOPATH=/gopath -e GOARCH=arm64 -v $(GOLANG_DOCKER_GOCACHE)/arm64:/root/.cache/go-build -v $(GOLANG_DOCKER_GOPATH):/gopath -v $(shell pwd):/ooni -w /ooni $(GOLANG_DOCKER_IMAGE) ./CLI/linux/build -tags=netgo,$(OONI_PSIPHON_TAGS) $(GOLANG_EXTRA_FLAGS)

#help:
#help: The `./mk ./CLI/ooniprobe/windows` command builds the ooniprobe official
#help: command line client for windows/386 and windows/amd64. This entails
#help: building and PGP signing the executables.
#help:
#help: You can also build the following subtargets:
.PHONY: ./CLI/ooniprobe/windows
./CLI/ooniprobe/windows:                   \
	./ooniprobe_windows_386.tar.gz.asc     \
	./ooniprobe_windows_386.zip.asc        \
	./ooniprobe_windows_amd64.tar.gz.asc   \
	./ooniprobe_windows_amd64.zip.asc

# ./ooniprobe_windows_386.tar.gz.asc creates and signs the release tarball
.PHONY:   ./ooniprobe_windows_386.tar.gz.asc
./ooniprobe_windows_386.tar.gz.asc: ./CLI/windows/386/ooniprobe.exe
	rm -f ooniprobe_windows_386.tar.gz ooniprobe_windows_386.tar.gz.asc
	tar -cvzf ooniprobe_windows_386.tar.gz -C ./CLI/windows/386 ooniprobe.exe
	gpg -abu $(GPG_USER) ooniprobe_windows_386.tar.gz

# ./ooniprobe_windows_386.zip.asc creates and signs the release zipball
.PHONY:   ./ooniprobe_windows_386.zip.asc
./ooniprobe_windows_386.zip.asc: ./CLI/windows/386/ooniprobe.exe
	rm -f ooniprobe_windows_386.zip ooniprobe_windows_386.zip.asc
	cd ./CLI/windows/386 && zip ../../../ooniprobe_windows_386.zip ooniprobe.exe
	gpg -abu $(GPG_USER) ooniprobe_windows_386.zip

#help:
#help: * `./mk ./CLI/windows/386/ooniprobe.exe`: windows/386
.PHONY:     ./CLI/windows/386/ooniprobe.exe
./CLI/windows/386/ooniprobe.exe: search/for/go search/for/mingw-w64 maybe/copypsiphon
	GOOS=windows GOARCH=386 CGO_ENABLED=1 CC=i686-w64-mingw32-gcc go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GOLANG_EXTRA_FLAGS) -o $@ ./cmd/ooniprobe

# ./ooniprobe_windows_amd64.tar.gz.asc creates and signs the release tarball
.PHONY:   ./ooniprobe_windows_amd64.tar.gz.asc
./ooniprobe_windows_amd64.tar.gz.asc: ./CLI/windows/amd64/ooniprobe.exe
	rm -f ooniprobe_windows_amd64.tar.gz ooniprobe_windows_amd64.tar.gz.asc
	tar -cvzf ooniprobe_windows_amd64.tar.gz -C ./CLI/windows/amd64 ooniprobe.exe
	gpg -abu $(GPG_USER) ooniprobe_windows_amd64.tar.gz

# ./ooniprobe_windows_amd64.zip.asc creates and signs the release zipball
.PHONY:   ./ooniprobe_windows_amd64.zip.asc
./ooniprobe_windows_amd64.zip.asc: ./CLI/windows/amd64/ooniprobe.exe
	rm -f ooniprobe_windows_amd64.zip ooniprobe_windows_amd64.zip.asc
	cd ./CLI/windows/amd64 && zip ../../../ooniprobe_windows_amd64.zip ooniprobe.exe
	gpg -abu $(GPG_USER) ooniprobe_windows_amd64.zip

#help:
#help: * `./mk ./CLI/windows/amd64/ooniprobe.exe`: windows/amd64
.PHONY:     ./CLI/windows/amd64/ooniprobe.exe
./CLI/windows/amd64/ooniprobe.exe: search/for/go search/for/mingw-w64 maybe/copypsiphon
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GOLANG_EXTRA_FLAGS) -o $@ ./cmd/ooniprobe

#help:
#help: The `./mk ./MOBILE/android` command builds the oonimkall library for Android.
#help:
#help: You can also build the following subtargets:
.PHONY: ./MOBILE/android
./MOBILE/android: search/for/gpg search/for/jar ./MOBILE/android/oonimkall.aar
	cp ./MOBILE/android/oonimkall.aar ./MOBILE/android/oonimkall-$(OONIMKALL_V).aar
	cp ./MOBILE/android/oonimkall-sources.jar ./MOBILE/android/oonimkall-$(OONIMKALL_V)-sources.jar
	cat ./MOBILE/template.pom | sed -e "s/@VERSION@/$(OONIMKALL_V)/g" > ./MOBILE/android/oonimkall-$(OONIMKALL_V).pom
	gpg -abu $(GPG_USER) ./MOBILE/android/oonimkall-$(OONIMKALL_V).aar
	gpg -abu $(GPG_USER) ./MOBILE/android/oonimkall-$(OONIMKALL_V)-sources.jar
	gpg -abu $(GPG_USER) ./MOBILE/android/oonimkall-$(OONIMKALL_V).pom
	cd ./MOBILE/android && jar -cf bundle.jar oonimkall-$(OONIMKALL_V).aar oonimkall-$(OONIMKALL_V).aar.asc oonimkall-$(OONIMKALL_V)-sources.jar oonimkall-$(OONIMKALL_V)-sources.jar.asc oonimkall-$(OONIMKALL_V).pom oonimkall-$(OONIMKALL_V).pom.asc

#help:
#help: * `./mk ./MOBILE/android/oonimkall.aar`: the AAR
.PHONY:   ./MOBILE/android/oonimkall.aar
./MOBILE/android/oonimkall.aar: android/sdk ooni/go maybe/copypsiphon
	PATH=$(OONIGODIR)/bin:$$PATH $(MAKE) -f mk __android_build_with_ooni_go

# GOMOBILE is the full path location to the gomobile binary. We want to
# execute this command every time, because its output may depend on context,
# for this reason WE ARE NOT using `:=`.
GOMOBILE = $(shell go env GOPATH)/bin/gomobile

# Here we use ooni/go to work around https://github.com/ooni/probe/issues/1444
__android_build_with_ooni_go: search/for/go
	go get -u golang.org/x/mobile/cmd/gomobile
	$(GOMOBILE) init
	PATH=$(shell go env GOPATH)/bin:$$PATH ANDROID_HOME=$(OONI_ANDROID_HOME) ANDROID_NDK_HOME=$(OONI_ANDROID_HOME)/ndk/$(ANDROID_NDK_VERSION) $(GOMOBILE) bind -target android -o ./MOBILE/android/oonimkall.aar -tags="$(OONI_PSIPHON_TAGS)" -ldflags '-s -w' $(GOLANG_EXTRA_FLAGS) ./pkg/oonimkall

#help:
#help: The `./mk ./MOBILE/ios` command builds the oonimkall library for iOS.
#help:
#help: You can also build the following subtargets:
.PHONY: ./MOBILE/ios
./MOBILE/ios:                             \
	./MOBILE/ios/oonimkall.framework.zip  \
	./MOBILE/ios/oonimkall.podspec

#help:
#help: * `./mk ./MOBILE/ios/oonimkall.framework.zip`: zip the framework
.PHONY:   ./MOBILE/ios/oonimkall.framework.zip
./MOBILE/ios/oonimkall.framework.zip: search/for/zip ./MOBILE/ios/oonimkall.framework
	cd ./MOBILE/ios && rm -rf oonimkall.framework.zip
	cd ./MOBILE/ios && zip -yr oonimkall.framework.zip oonimkall.framework

#help:
#help: * `./mk ./MOBILE/ios/framework`: the framework
.PHONY:     ./MOBILE/ios/oonimkall.framework
./MOBILE/ios/oonimkall.framework: search/for/go search/for/xcode maybe/copypsiphon
	go get -u golang.org/x/mobile/cmd/gomobile
	$(GOMOBILE) init
	PATH=$(shell go env GOPATH)/bin:$$PATH $(GOMOBILE) bind -target ios -o $@ -tags="$(OONI_PSIPHON_TAGS)" -ldflags '-s -w' $(GOLANG_EXTRA_FLAGS) ./pkg/oonimkall

#help:
#help: * `./mk ./MOBILE/ios/oonimkall.podspec`: the podspec
.PHONY:   ./MOBILE/ios/oonimkall.podspec
./MOBILE/ios/oonimkall.podspec: ./MOBILE/template.podspec
	cat $< | sed -e "s/@VERSION@/$(OONIMKALL_V)/g" -e "s/@RELEASE@/$(OONIMKALL_R)/g" > $@

# important: OONIMKALL_V and OONIMKALL_R MUST be expanded just once so we use `:=`
OONIMKALL_V := $(shell date -u +%Y.%m.%d-%H%M%S)
OONIMKALL_R := $(shell git describe --tags || echo '0.0.0-dev')

#help: The `debian/publish` target publishes all the debian packages
#help: present in the toplevel directory using debopos-ci.
# TODO(bassosimone): do not hardcode using linux/amd64 here?
.PHONY: debian/publish
debian/publish: search/for/docker
	test -z "$(CI)" || { echo "fatal: refusing to run in a CI environment" 1>&2; exit 1; }
	ls *.deb 2>/dev/null || { echo "fatal: no debian packages in the toplevel dir" 1>&2; exit 1; }
	test -n "$(AWS_ACCESS_KEY_ID)" || { echo "fatal: AWS_ACCESS_KEY_ID not set" 1>&2; exit 1; }
	test -n "$(AWS_SECRET_ACCESS_KEY)" || { echo "fatal: AWS_SECRET_ACCESS_KEY not set" 1>&2; exit 1; }
	test -n "$(DEB_GPG_KEY)" || { echo "fatal: DEB_GPG_KEY not set" 1>&2; exit 1; }
	docker pull --platform linux/amd64 ubuntu:20.04
	docker run --platform linux/amd64 -e AWS_ACCESS_KEY_ID="$(AWS_ACCESS_KEY_ID)" -e AWS_SECRET_ACCESS_KEY="$(AWS_SECRET_ACCESS_KEY)" -e DEB_GPG_KEY="$(DEB_GPG_KEY)" -v $(shell pwd):/ooni -w /ooni ubuntu:20.04 ./CLI/linux/pubdebian

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

#help:
#help: * `./mk search/for/gpg`: checks for gpg
.PHONY: search/for/gpg
search/for/gpg:
	@printf "checking for gpg... "
	@command -v gpg || { echo "not found"; exit 1; }

#help:
#help: * `./mk search/for/go`: checks for go
.PHONY: search/for/go
search/for/go:
	@printf "checking for go... "
	@command -v go || { echo "not found"; exit 1; }
	@printf "checking for go version... "
	@echo $(__GOVERSION_REAL)
	@[ "$(GOLANG_VERSION_STRING)" = "$(__GOVERSION_REAL)" ] || { echo "fatal: go version must be $(GOLANG_VERSION_STRING) instead of $(__GOVERSION_REAL)"; exit 1; }

# __GOVERSION_REAL is the go version reported by the go binary (we
# SHOULD NOT cache this value so we ARE NOT using `:=`)
__GOVERSION_REAL = $(shell go version | awk '{print $$3}')

#help:
#help: * `./mk search/for/jar`: checks for jar
.PHONY: search/for/jar
search/for/jar:
	@printf "checking for jar... "
	@command -v jar || { echo "not found"; exit 1; }

#help:
#help: * `./mk search/for/java`: checks for java
.PHONY: search/for/java
search/for/java:
	@printf "checking for java... "
	@command -v java || { echo "not found"; exit 1; }

#help:
#help: * `./mk search/for/mingw-w64`: checks for mingw-w64
.PHONY: search/for/mingw-w64
search/for/mingw-w64:
	@printf "checking for x86_64-w64-mingw32-gcc... "
	@command -v x86_64-w64-mingw32-gcc || { echo "not found"; exit 1; }
	@printf "checking for x86_64-w64-mingw32-gcc version... "
	@echo $(__MINGW32_AMD64_VERSION)
	@[ "$(MINGW_W64_VERSION)" = "$(__MINGW32_AMD64_VERSION)" ] || { echo "fatal: x86_64-w64-mingw32-gcc version must be $(MINGW_W64_VERSION) instead of $(__MINGW32_AMD64_VERSION)"; exit 1; }
	@printf "checking for i686-w64-mingw32-gcc... "
	@command -v i686-w64-mingw32-gcc || { echo "not found"; exit 1; }
	@printf "checking for i686-w64-mingw32-gcc version... "
	@echo $(__MINGW32_386_VERSION)
	@[ "$(MINGW_W64_VERSION)" = "$(__MINGW32_386_VERSION)" ] || { echo "fatal: i686-w64-mingw32-gcc version must be $(MINGW_W64_VERSION) instead of $(__MINGW32_386_VERSION)"; exit 1; }

# __MINGW32_AMD64_VERSION and __MINGW32_386_VERSION are the versions
# reported by the amd64 and 386 mingw binaries.
__MINGW32_AMD64_VERSION = $(shell x86_64-w64-mingw32-gcc --version | sed -n 1p | awk '{print $$3}')
__MINGW32_386_VERSION = $(shell i686-w64-mingw32-gcc --version | sed -n 1p | awk '{print $$3}')

#help:
#help: * `./mk search/for/shasum`: checks for shasum
.PHONY: search/for/shasum
search/for/shasum:
	@printf "checking for shasum... "
	@command -v shasum || { echo "not found"; exit 1; }

#help:
#help: * `./mk search/for/xcode`: checks for Xcode
.PHONY: search/for/xcode
search/for/xcode:
	@printf "checking for xcodebuild... "
	@command -v xcodebuild || { echo "not found"; exit 1; }
	@printf "checking for Xcode version... "
	@echo $(__XCODEVERSION_REAL)
	@[ "$(XCODE_VERSION)" = "$(__XCODEVERSION_REAL)" ] || { echo "fatal: Xcode version must be $(XCODE_VERSION) instead of $(__XCODEVERSION_REAL)"; exit 1; }

# __XCODEVERSION_REAL is the version of Xcode obtained using xcodebuild
__XCODEVERSION_REAL = `xcodebuild -version | grep ^Xcode | awk '{print $$2}'`

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
#help: The `./mk maybe/copypsiphon` command copies the private psiphon config
#help: file into the current tree unless `$(OONI_PSIPHON_TAGS)` is empty.
.PHONY: maybe/copypsiphon
maybe/copypsiphon: search/for/git
	test -z "$(OONI_PSIPHON_TAGS)" || $(MAKE) -f mk $(OONIPRIVATE)
	test -z "$(OONI_PSIPHON_TAGS)" || cp $(OONIPRIVATE)/psiphon-config.key ./internal/engine
	test -z "$(OONI_PSIPHON_TAGS)" || cp $(OONIPRIVATE)/psiphon-config.json.age ./internal/engine

# OONIPRIVATE is the directory where we clone the private repository.
OONIPRIVATE = $(GIT_CLONE_DIR)/github.com/ooni/probe-private

# OONIPRIVATE_REPO is the private repository URL.
OONIPRIVATE_REPO = git@github.com:ooni/probe-private

# $(OONIPRIVATE) clones the private repository in $(GIT_CLONE_DIR)
$(OONIPRIVATE): search/for/git $(GIT_CLONE_DIR)
	test -d $(OONIPRIVATE) || $(MAKE) -f mk __really_clone_private_repo

__really_clone_private_repo:
	git clone $(OONIPRIVATE_REPO) $(OONIPRIVATE)

#help:
#help: The `./mk ooni/go` command builds the latest version of ooni/go.
.PHONY: ooni/go
ooni/go: search/for/bash search/for/git search/for/go $(OONIGODIR)
	test -d $(OONIGODIR) || git clone -b ooni --single-branch --depth 8 $(OONIGO_REPO) $(OONIGODIR)
	cd $(OONIGODIR) && git pull --ff-only
	cd $(OONIGODIR)/src && ./make.bash

# OONIGODIR is the directory in which we clone ooni/go
OONIGODIR = $(GIT_CLONE_DIR)/github.com/ooni/go

# OONIGO_REPO is the repository for ooni/go
OONIGO_REPO = https://github.com/ooni/go

#help:
#help: The `./mk android/sdk` command ensures we are using the
#help: correct version of the Android sdk.
.PHONY: android/sdk
android/sdk: search/for/java
	test -d $(OONI_ANDROID_HOME) || $(MAKE) -f mk android/sdk/download
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

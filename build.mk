# The following variables control the versions of the tools that
# we use thoughout this (GNU) makefile. We need to ensure these
# tools are up-to-date as part of the release process.

# ANDROID_CLITOOLS_VERSION is the version of the Android CLI tools version.
ANDROID_CLITOOLS_VERSION = 7302050

# ANDROID_CLITOOLS_SHA256 is the SHA256 of the CLI tools file.
ANDROID_CLITOOLS_SHA256 = 7a00faadc0864f78edd8f4908a629a46d622375cbe2e5814e82934aebecdb622

# ANDROID_NDK_VERSION is the Android NDK version.
ANDROID_NDK_VERSION = 22.1.7171670

# ANDROID_INSTALL_EXTRA is the extra stuff we need to install.
ANDROID_INSTALL_EXTRA = 'build-tools;29.0.3' 'platforms;android-30'

# __GOVERSION, GOVERSION, and GODOCKER identify the Go version we expect.
__GOVERSION = 1.16.4
GOVERSION = go$(__GOVERSION)
GODOCKER = golang:$(__GOVERSION)-alpine

# MINGW64_VERSION contains the mingw-w64 version
MINGW64_VERSION = 10.3.1

# XCODEVERSION is the version of Xcode we expect
XCODEVERSION = 12.5

# The rest of this makefile defines the available targets. Most of
# them are documented using `#quickhelp:` or `#help:` descriptors that
# cause the comments to appear when running `./mk help`.

#quickhelp: Usage: ./mk [VARIABLE=VALUE ...] TARGET ...
.PHONY: quickhelp
quickhelp:
	@cat build.mk | grep '^#quickhelp:' | sed -e 's/^#quickhelp://' -e 's/^\ *//'

#quickhelp:
#quickhelp: The `./mk printtargets` command prints all available targets.
.PHONY: printtargets
printtargets:
	@cat build.mk | grep '^\.PHONY:' | sed -e 's/^\.PHONY://' -e 's/^/*/'

#quickhelp:
#quickhelp: The `./mk help` command provides detailed usage instructions. We
#quickhelp: recommend running `./mk help|less` to page the output.
.PHONY: help
help:
	@cat build.mk | grep -E '^#(quick)?help:' | sed -E -e 's/^#(quick)?help://' -e s'/^\ //'

#help:
#help: The following variables control the build. You can specify them
#help: before the targets as indicated above in the usage line.

#help:
#help: * GIT_CLONE_DIR       : directory where to clone repositories, by default
#help:                         set to `$HOME/.ooniprobe-build/src`.
GIT_CLONE_DIR = $(HOME)/.ooniprobe-build/src

$(GIT_CLONE_DIR):
	mkdir -p $(GIT_CLONE_DIR)

#help:
#help: * GO_EXTRA_FLAGS      : extra flags passed to `go build ...`, empty by
#help:                         default. Useful to pass flags to `go`, e.g.:
#help:
#help:                             ./mk GO_EXTRA_FLAGS="-x -v" miniooni
GO_EXTRA_FLAGS =

#help:
#help: * GPG_USER            : allows overriding the default GPG user used
#help:                         to sign binary releases, e.g.:
#help:
#help:                             ./mk GPG_USER=john@doe.com ooniprobe/windows
GPG_USER = simone@openobservatory.org

#help:
#help: * OONI_PSIPHON_TAGS   : build tags for `go build -tags ...` that cause
#help:                         the build to embed a psiphon configuration file
#help:                         into the generated binaries. This build tag
#help:                         implies cloning the git@github.com:ooni/probe-private
#help:                         repository. If you do not have the permission to
#help:                         clone ooni-private just clear this variable, e.g.:
#help:
#help:                             ./mk OONI_PSIPHON_TAGS="" miniooni
OONI_PSIPHON_TAGS = ooni_psiphon_config

#help:
#help: * OONI_ANDROID_HOME   : directory where the Android SDK is downloaded
#help:                         and installed. You can point this to an existing
#help:                         copy of the SDK as long as (1) you have the
#help:                         right version of the command line tools, and
#help:                         (2) it's okay for us to install packages.
OONI_ANDROID_HOME = $(HOME)/.ooniprobe-build/sdk/android

#help:
#help: The `./mk printvars` command prints the current value of the above
#help: listed build-controlling variables.
.PHONY: printvars
printvars:
	@echo "GIT_CLONE_DIR=$(GIT_CLONE_DIR)"
	@echo "GO_EXTRA_FLAGS=$(GO_EXTRA_FLAGS)"
	@echo "OONI_PSIPHON_TAGS=$(OONI_PSIPHON_TAGS)"

#help:
#help: The `./mk miniooni` command builds the miniooni experimental
#help: command line client for all the supported GOOS/GOARCH.
#help:
#help: We also support the following commands:
.PHONY: miniooni
miniooni:                             \
	./CLI/darwin/amd64/miniooni       \
	./CLI/darwin/arm64/miniooni       \
	./CLI/linux/386/miniooni          \
	./CLI/linux/amd64/miniooni        \
	./CLI/linux/arm/miniooni          \
	./CLI/linux/arm64/miniooni        \
	./CLI/windows/386/miniooni.exe    \
	./CLI/windows/amd64/miniooni.exe

#help:
#help: * `./mk ./CLI/darwin/amd64/miniooni`: darwin/amd64
.PHONY: ./CLI/darwin/amd64/miniooni
./CLI/darwin/amd64/miniooni: command/go maybe/copypsiphon
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GO_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/darwin/arm64/miniooni`: darwin/arm64
.PHONY: ./CLI/darwin/arm64/miniooni
./CLI/darwin/arm64/miniooni: command/go maybe/copypsiphon
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GO_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/linux/386/miniooni`: linux/386
.PHONY: ./CLI/linux/386/miniooni
./CLI/linux/386/miniooni: command/go maybe/copypsiphon
	GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -tags="netgo,$(OONI_PSIPHON_TAGS)" -ldflags="-s -w -extldflags -static" $(GO_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/linux/amd64/miniooni`: linux/amd64
.PHONY: ./CLI/linux/amd64/miniooni
./CLI/linux/amd64/miniooni: command/go maybe/copypsiphon
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags="netgo,$(OONI_PSIPHON_TAGS)" -ldflags="-s -w -extldflags -static" $(GO_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/linux/arm/miniooni`: linux/arm
.PHONY: ./CLI/linux/arm/miniooni
./CLI/linux/arm/miniooni: command/go maybe/copypsiphon
	GOOS=linux GOARCH=arm CGO_ENABLED=0 GOARM=7 go build -tags="netgo,$(OONI_PSIPHON_TAGS)" -ldflags="-s -w -extldflags -static" $(GO_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/linux/arm64/miniooni`: linux/arm64
.PHONY: ./CLI/linux/arm64/miniooni
./CLI/linux/arm64/miniooni: command/go maybe/copypsiphon
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags="netgo,$(OONI_PSIPHON_TAGS)" -ldflags="-s -w -extldflags -static" $(GO_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/windows/386/miniooni.exe`: windows/386
.PHONY: ./CLI/windows/386/miniooni.exe
./CLI/windows/386/miniooni.exe: command/go maybe/copypsiphon
	GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GO_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `./mk ./CLI/windows/amd64/miniooni.exe`: windows/amd64
.PHONY: ./CLI/windows/amd64/miniooni.exe
./CLI/windows/amd64/miniooni.exe: command/go maybe/copypsiphon
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GO_EXTRA_FLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: The `./mk ooniprobe/darwin` command builds the ooniprobe official
#help: command line client for darwin/amd64 and darwin/arm64.
#help:
#help: We also support the following commands:
.PHONY: ooniprobe/darwin
ooniprobe/darwin:                    \
	./CLI/darwin/amd64/ooniprobe.asc \
	./CLI/darwin/arm64/ooniprobe.asc

.PHONY: ./CLI/darwin/amd64/ooniprobe.asc
./CLI/darwin/amd64/ooniprobe.asc: ./CLI/darwin/amd64/ooniprobe
	rm -f $@ && gpg -abu $(GPG_USER) $<

#help:
#help: * `./mk ./CLI/darwin/amd64/ooniprobe`: darwin/amd64
.PHONY: ./CLI/darwin/amd64/ooniprobe
./CLI/darwin/amd64/ooniprobe: command/go maybe/copypsiphon
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GO_EXTRA_FLAGS) -o $@ ./cmd/ooniprobe

.PHONY: ./CLI/darwin/arm64/ooniprobe.asc
./CLI/darwin/arm64/ooniprobe.asc: ./CLI/darwin/arm64/ooniprobe
	rm -f $@ && gpg -abu $(GPG_USER) $<

#help:
#help: * `./mk ./CLI/darwin/arm64/ooniprobe`: darwin/arm64
.PHONY: ./CLI/darwin/arm64/ooniprobe
./CLI/darwin/arm64/ooniprobe: command/go maybe/copypsiphon
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GO_EXTRA_FLAGS) -o $@ ./cmd/ooniprobe

#help:
#help: The `./mk ooniprobe/debian` command builds the ooniprobe CLI
#help: debian package for amd64 and arm64.
#help:
#help: We also support the following commands:
.PHONY: ooniprobe/debian
ooniprobe/debian:           \
	ooniprobe/debian/amd64  \
	ooniprobe/debian/arm64

#help:
#help: * `./mk ooniprobe/debian/amd64`: debian/amd64
.PHONY: ooniprobe/debian/amd64
ooniprobe/debian/amd64: command/docker ./CLI/linux/amd64/ooniprobe
	docker pull --platform linux/amd64 debian:stable
	docker run --platform linux/amd64 -v `pwd`:/ooni -w /ooni debian:stable ./CLI/linux/debian

#help:
#help: * `./mk ooniprobe/debian/arm64`: debian/arm64
.PHONY: ooniprobe/debian/arm64
ooniprobe/debian/arm64: command/docker ./CLI/linux/arm64/ooniprobe
	docker pull --platform linux/arm64 debian:stable
	docker run --platform linux/arm64 -v `pwd`:/ooni -w /ooni debian:stable ./CLI/linux/debian

#help:
#help: The `./mk ooniprobe/linux` command builds the ooniprobe official command
#help: line client for amd64 and arm64.
#help:
#help: We also support the following commands:
.PHONY: ooniprobe/linux
ooniprobe/linux:                     \
	./CLI/linux/amd64/ooniprobe.asc  \
	./CLI/linux/arm64/ooniprobe.asc

.PHONY: ./CLI/linux/amd64/ooniprobe.asc
./CLI/linux/amd64/ooniprobe.asc: ./CLI/linux/amd64/ooniprobe
	rm -f $@ && gpg -abu $(GPG_USER) $<

#help:
#help: * `./mk ./CLI/linux/amd64/ooniprobe`: linux/amd64
.PHONY: ./CLI/linux/amd64/ooniprobe
./CLI/linux/amd64/ooniprobe: command/docker maybe/copypsiphon
	docker pull --platform linux/amd64 $(GODOCKER)
	docker run --platform linux/amd64 -e GOARCH=amd64 -v `pwd`:/ooni -w /ooni $(GODOCKER) ./CLI/linux/build -tags=netgo,$(OONI_PSIPHON_TAGS)

.PHONY: ./CLI/linux/arm64/ooniprobe.asc
./CLI/linux/arm64/ooniprobe.asc: ./CLI/linux/arm64/ooniprobe
	rm -f $@ && gpg -abu $(GPG_USER) $<

#help:
#help: * `./mk ./CLI/linux/arm64/ooniprobe`: linux/arm64
.PHONY: ./CLI/linux/arm64/ooniprobe
./CLI/linux/arm64/ooniprobe: command/docker maybe/copypsiphon
	docker pull --platform linux/arm64 $(GODOCKER)
	docker run --platform linux/arm64 -e GOARCH=arm64 -v `pwd`:/ooni -w /ooni $(GODOCKER) ./CLI/linux/build -tags=netgo,$(OONI_PSIPHON_TAGS)

#help:
#help: The `./mk ooniprobe/windows` command builds the ooniprobe official
#help: command line client for windows/386 and windows/amd64.
#help:
#help: We also support the following commands:
.PHONY: ooniprobe/windows
ooniprobe/windows:                         \
	./CLI/windows/386/ooniprobe.exe.asc    \
	./CLI/windows/amd64/ooniprobe.exe.asc

.PHONY: ./CLI/windows/386/ooniprobe.exe.asc
./CLI/windows/386/ooniprobe.exe.asc: ./CLI/windows/386/ooniprobe.exe
	rm -f $@ && gpg -abu $(GPG_USER) $<

#help:
#help: * `./mk ./CLI/windows/386/ooniprobe.exe`: windows/386
.PHONY: ./CLI/windows/386/ooniprobe.exe
./CLI/windows/386/ooniprobe.exe: command/go command/mingw-w64 maybe/copypsiphon
	GOOS=windows GOARCH=386 CGO_ENABLED=1 CC=i686-w64-mingw32-gcc go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GO_EXTRA_FLAGS) -o $@ ./cmd/ooniprobe

.PHONY: ./CLI/windows/amd64/ooniprobe.exe.asc
./CLI/windows/amd64/ooniprobe.exe.asc: ./CLI/windows/amd64/ooniprobe.exe
	rm -f $@ && gpg -abu $(GPG_USER) $<

#help:
#help: * `./mk ./CLI/windows/amd64/ooniprobe.exe`: windows/amd64
.PHONY: ./CLI/windows/amd64/ooniprobe.exe
./CLI/windows/amd64/ooniprobe.exe: command/go command/mingw-w64 maybe/copypsiphon
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -tags="$(OONI_PSIPHON_TAGS)" -ldflags="-s -w" $(GO_EXTRA_FLAGS) -o $@ ./cmd/ooniprobe

#help:
#help: The `./mk android` command builds the oonimkall library for Android.
#help:
#help: We also support the following commands:
.PHONY: android
android: command/gpg command/jar ./MOBILE/android/oonimkall.aar

#help:
#help: * `./mk ./MOBILE/android/oonimkall.aar`: just the AAR
.PHONY: ./MOBILE/android/oonimkall.aar
./MOBILE/android/oonimkall.aar: android/sdk ooni/go
	PATH=$(OONIGODIR)/bin:$$PATH $(MAKE) -f build.mk __android_build_with_ooni_go

__android_build_with_ooni_go: command/go
	go get -u golang.org/x/mobile/cmd/gomobile@latest
	$(GOMOBILE) init
	ANDROID_HOME=$(OONI_ANDROID_HOME) ANDROID_NDK_HOME=$(OONI_ANDROID_HOME)/ndk/$(ANDROID_NDK_VERSION) $(GOMOBILE) bind -target android -o ./MOBILE/android/oonimkall.aar -tags="$(OONI_PSIPHON_TAGS)" -ldflags '-s -w' ./pkg/oonimkall

#help:
#help: The `./mk ios` command builds the oonimkall library for iOS.
#help:
#help: We also support the following commands:
.PHONY: ios
ios:                                      \
	./MOBILE/ios/oonimkall.framework.zip  \
	./MOBILE/ios/oonimkall.podspec

#help:
#help: * `./mk ./MOBILE/ios/oonimkall.framework.zip`: zip the framework
.PHONY: ./MOBILE/ios/oonimkall.framework.zip
./MOBILE/ios/oonimkall.framework.zip: command/zip ./MOBILE/ios/oonimkall.framework
	cd ./MOBILE/ios && rm -rf oonimkall.framework.zip
	cd ./MOBILE/ios && zip -yr oonimkall.framework.zip oonimkall.framework

#help:
#help: * `./mk ./MOBILE/ios/framework`: the framework
.PHONY: ./MOBILE/ios/oonimkall.framework
./MOBILE/ios/oonimkall.framework: command/go command/xcode
	go get -u golang.org/x/mobile/cmd/gomobile@latest
	$(GOMOBILE) init
	$(GOMOBILE) bind -target ios -o $@ -tags="$(OONI_PSIPHON_TAGS)" -ldflags '-s -w' ./pkg/oonimkall

GOMOBILE = `go env GOPATH`/bin/gomobile

#help:
#help: * `./mk ./MOBILE/ios/oonimkall.podspec`: the podspec
./MOBILE/ios/oonimkall.podspec: ./MOBILE/template.podspec
	cat $< | sed -e 's/@VERSION@/$(OONIMKALL_V)/g' -e 's/@RELEASE@/$(OONIMKALL_R)/g' > $@

OONIMKALL_V = `date -u +%Y.%m.%d-%H%M%S`
OONIMKALL_R = `git describe --tags`

#help:
#help: The following commands check for the availability of dependencies:

#help:
#help: * `./mk command/bash`: checks for bash
.PHONY: command/bash
command/bash:
	@printf "checking for bash... "
	@command -v bash || { echo "not found"; exit 1; }

#help:
#help: * `./mk command/curl`: checks for curl
.PHONY: command/curl
command/curl:
	@printf "checking for curl... "
	@command -v curl || { echo "not found"; exit 1; }

#help:
#help: * `./mk command/docker`: checks for docker
.PHONY: command/docker
command/docker:
	@printf "checking for docker... "
	@command -v git || { echo "not found"; exit 1; }

#help:
#help: * `./mk command/git`: checks for git
.PHONY: command/git
command/git:
	@printf "checking for git... "
	@command -v git || { echo "not found"; exit 1; }

#help:
#help: * `./mk command/jar`: checks for jar
.PHONY: command/jar
command/jar:
	@printf "checking for jar... "
	@command -v jar || { echo "not found"; exit 1; }

#help:
#help: * `./mk command/java`: checks for java
.PHONY: command/java
command/java:
	@printf "checking for java... "
	@command -v java || { echo "not found"; exit 1; }

#help:
#help: * `./mk command/go`: checks for go
.PHONY: command/go
command/go:
	@printf "checking for go... "
	@command -v go || { echo "not found"; exit 1; }
	@printf "checking for go version... "
	@echo $(__GOVERSION_REAL)
	@[ "$(GOVERSION)" = "$(__GOVERSION_REAL)" ] || { echo "fatal: go version must be $(GOVERSION) instead of $(__GOVERSION_REAL)"; exit 1; }

# $(__GOVERSION_REAL) is the Go version according to the `go` executable.
__GOVERSION_REAL=$$(go version | awk '{print $$3}')

#help:
#help: * `./mk command/mingw-w64`: checks for mingw-w64
.PHONY: command/mingw-w64
command/mingw-w64:
	@printf "checking for x86_64-w64-mingw32-gcc... "
	@command -v x86_64-w64-mingw32-gcc || { echo "not found"; exit 1; }
	@printf "checking for x86_64-w64-mingw32-gcc version... "
	@echo $(__MINGW32_AMD64_VERSION)
	@[ "$(MINGW64_VERSION)" = "$(__MINGW32_AMD64_VERSION)" ] || { echo "fatal: x86_64-w64-mingw32-gcc version must be $(MINGW64_VERSION) instead of $(__MINGW32_AMD64_VERSION)"; exit 1; }
	@printf "checking for i686-w64-mingw32-gcc... "
	@command -v i686-w64-mingw32-gcc || { echo "not found"; exit 1; }
	@printf "checking for i686-w64-mingw32-gcc version... "
	@echo $(__MINGW32_386_VERSION)
	@[ "$(MINGW64_VERSION)" = "$(__MINGW32_386_VERSION)" ] || { echo "fatal: i686-w64-mingw32-gcc version must be $(MINGW64_VERSION) instead of $(__MINGW32_386_VERSION)"; exit 1; }

__MINGW32_AMD64_VERSION = `x86_64-w64-mingw32-gcc --version | sed -n 1p | awk '{print $$3}'`
__MINGW32_386_VERSION = `i686-w64-mingw32-gcc --version | sed -n 1p | awk '{print $$3}'`

#help:
#help: * `./mk command/shasum`: checks for shasum
.PHONY: command/shasum
command/shasum:
	@printf "checking for shasum... "
	@command -v shasum || { echo "not found"; exit 1; }

#help:
#help: * `./mk command/xcode`: checks for Xcode
.PHONY: command/xcode
command/xcode:
	@printf "checking for xcodebuild... "
	@command -v xcodebuild || { echo "not found"; exit 1; }
	@printf "checking for Xcode version... "
	@echo $(__XCODEVERSION_REAL)
	@[ "$(XCODEVERSION)" = "$(__XCODEVERSION_REAL)" ] || { echo "fatal: Xcode version must be $(XCODEVERSION) instead of $(__XCODEVERSION_REAL)"; exit 1; }

__XCODEVERSION_REAL = `xcodebuild -version | grep ^Xcode | awk '{print $$2}'`

#help:
#help: * `./mk command/unzip`: checks for unzip
.PHONY: command/unzip
command/unzip:
	@printf "checking for unzip... "
	@command -v unzip || { echo "not found"; exit 1; }

#help:
#help: * `./mk command/zip`: checks for zip
.PHONY: command/zip
command/zip:
	@printf "checking for zip... "
	@command -v zip || { echo "not found"; exit 1; }

#help:
#help: The `./mk maybe/copypsiphon` command copies the private psiphon config
#help: file into the current tree unless `$(OONI_PSIPHON_TAGS)` is empty.
.PHONY: maybe/copypsiphon
maybe/copypsiphon: command/git
	test -z "$(OONI_PSIPHON_TAGS)" || $(MAKE) -f build.mk $(OONIPRIVATE)
	test -z "$(OONI_PSIPHON_TAGS)" || cp $(OONIPRIVATE)/psiphon-config.key ./internal/engine
	test -z "$(OONI_PSIPHON_TAGS)" || cp $(OONIPRIVATE)/psiphon-config.json.age ./internal/engine

# OONIPRIVATE is the directory where we clone the private repository.
OONIPRIVATE = $(GIT_CLONE_DIR)/github.com/ooni/probe-private

# OONIPRIVATE_REPO is the private repository URL.
OONIPRIVATE_REPO = git@github.com:ooni/probe-private

# $(OONIPRIVATE) clones the private repository in $(GIT_CLONE_DIR)
$(OONIPRIVATE): command/git $(GIT_CLONE_DIR)
	git clone $(OONIPRIVATE_REPO) $(OONIPRIVATE)

#help:
#help: The `./mk ooni/go` command builds the latest version of ooni/go.
.PHONY: ooni/go
ooni/go: command/bash command/git command/go $(OONIGODIR)
	test -d $(OONIGODIR) || git clone -b ooni --single-branch --depth 8 $(OONIGO_REPO) $(OONIGODIR)
	cd $(OONIGODIR) && git pull
	cd $(OONIGODIR)/src && ./make.bash

# OONIGODIR is the directory in which we clone ooni/go
OONIGODIR = $(GIT_CLONE_DIR)/github.com/ooni/go

# OONIGO_REPO is the repository for ooni/go
OONIGO_REPO = https://github.com/ooni/go

#help:
#help: The `./mk android/sdk` command ensures we are using the
#help: correct version of the Android sdk.
.PHONY: android/sdk
android/sdk: command/java
	test -d $(OONI_ANDROID_HOME) || $(MAKE) -f build.mk android/sdk/download
	echo "Yes" | $(__ANDROID_SDKMANAGER) --install $(ANDROID_INSTALL_EXTRA) 'ndk;$(ANDROID_NDK_VERSION)'

__ANDROID_SDKMANAGER = $(OONI_ANDROID_HOME)/cmdline-tools/$(ANDROID_CLITOOLS_VERSION)/bin/sdkmanager

# See https://stackoverflow.com/a/61176718 to understand why
# we need to reorganize the directories like this:
#help:
#help: The `./mk android/sdk/download` unconditionally downloads the
#help: Android SDK at `$(OONI_ANDROID_HOME)`.
android/sdk/download: command/curl command/java command/shasum command/unzip
	curl -fsSLO https://dl.google.com/android/repository/$(__ANDROID_CLITOOLS_FILE)
	echo "$(ANDROID_CLITOOLS_SHA256)  $(__ANDROID_CLITOOLS_FILE)" > __SHA256
	shasum --check __SHA256
	rm -f __SHA256
	unzip $(__ANDROID_CLITOOLS_FILE)
	rm $(__ANDROID_CLITOOLS_FILE)
	mkdir -p $(OONI_ANDROID_HOME)/cmdline-tools
	mv cmdline-tools $(OONI_ANDROID_HOME)/cmdline-tools/$(ANDROID_CLITOOLS_VERSION)

__ANDROID_CLITOOLS_FILE = commandlinetools-linux-$(ANDROID_CLITOOLS_VERSION)_latest.zip

# Many rules in here break if run in parallel.
.NOTPARALLEL:

#quickhelp: Usage: make [VARIABLE=VALUE ...] TARGET ...
.PHONY: usage
usage:
	@cat Makefile | grep '^#quickhelp:' | sed -e 's/^#quickhelp://' -e 's/^\ *//'

# Most targets are .PHONY because whether to rebuild is controlled
# by golang. We expose to the user all the .PHONY targets.
#quickhelp:
#quickhelp: The `make list-targets` command lists all available targets.
.PHONY: list-targets
list-targets:
	@cat Makefile | grep '^\.PHONY:' | sed -e 's/^\.PHONY://' | grep -v '^ search' | grep -v '^ maybe'

#quickhelp:
#quickhelp: The `make help` command provides detailed usage instructions. We
#quickhelp: recommend running `make help|less` to page its output.
.PHONY: help
help:
	@cat Makefile | grep -E '^#(quick)?help:' | sed -E -e 's/^#(quick)?help://' -e s'/^\ //'

#help:
#help: The following variables control the build. You can specify them
#help: on the command line as a key-value pairs (see usage above).

#help:
#help: * GIT_CLONE_DIR           : directory where to clone repositories, by default
#help:                             set to `$HOME/.ooniprobe-build/src`.
GIT_CLONE_DIR = $(HOME)/.ooniprobe-build/src

#help:
#help: * OONI_GO_DOCKER_GOCACHE  : base directory to put GOMODCACHE and GOCACHE
#help:                             when building using Docker. By default this
#help:                             is set to `$HOME/.ooniprobe-build/cache`
#help:
OONI_GO_DOCKER_GOCACHE = $$(pwd)/GOCACHE

#help:
#help: * OONI_PSIPHON_TAGS       : build tags for `go build -tags ...` that cause
#help:                             the build to embed a psiphon configuration file
#help:                             into the generated binaries. This build tag
#help:                             implies cloning the git@github.com:ooni/probe-private
#help:                             repository. If you do not have the permission to
#help:                             clone it, just clear this variable, e.g.:
#help:
#help:                                 make OONI_PSIPHON_TAGS="" CLI/miniooni
OONI_PSIPHON_TAGS = ooni_psiphon_config

#quickhelp:
#quickhelp: The `make show-config` command shows the current value of the
#quickhelp: variables controlling the build.
.PHONY: show-config
show-config:
	@echo "GIT_CLONE_DIR=$(GIT_CLONE_DIR)"
	@echo "OONI_PSIPHON_TAGS=$(OONI_PSIPHON_TAGS)"

#help:
#help: The `make CLI/android-386` command builds miniooni and ooniprobe for android/386.
.PHONY: CLI/android-386
CLI/android-386: search/for/go search/for/android/sdk maybe/copypsiphon
	./CLI/go-build-android 386 ./internal/cmd/miniooni
	./CLI/go-build-android 386 ./cmd/ooniprobe

#help:
#help: The `make CLI/android-amd64` command builds miniooni and ooniprobe for android/amd64.
.PHONY: CLI/android-amd64
CLI/android-amd64: search/for/go search/for/android/sdk maybe/copypsiphon
	./CLI/go-build-android amd64 ./internal/cmd/miniooni
	./CLI/go-build-android amd64 ./cmd/ooniprobe

#help:
#help: The `make CLI/android-arm` command builds miniooni and ooniprobe for android/arm.
.PHONY: CLI/android-arm
CLI/android-arm: search/for/go search/for/android/sdk maybe/copypsiphon
	./CLI/go-build-android arm ./internal/cmd/miniooni
	./CLI/go-build-android arm ./cmd/ooniprobe

#help:
#help: The `make CLI/android-arm64` command builds miniooni and ooniprobe for android/arm64.
.PHONY: CLI/android-arm64
CLI/android-arm64: search/for/go search/for/android/sdk maybe/copypsiphon
	./CLI/go-build-android arm64 ./internal/cmd/miniooni
	./CLI/go-build-android arm64 ./cmd/ooniprobe

#help:
#help: The `make CLI/darwin` command builds the ooniprobe and miniooni
#help: command line clients for darwin/amd64 and darwin/arm64.
.PHONY: CLI/darwin
CLI/darwin:
	go run ./internal/cmd/buildtool darwin

#help:
#help: The `make CLI/linux-static-386` command builds and statically links the
#help: ooniprobe and miniooni binaries for linux/386.
.PHONY: CLI/linux-static-386
CLI/linux-static-386:
	go run ./internal/cmd/buildtool linux docker 386

#help:
#help: The `make CLI/linux-static-amd64` command builds and statically links the
#help: ooniprobe and miniooni binaries for linux/amd64.
.PHONY: CLI/linux-static-amd64
CLI/linux-static-amd64:
	go run ./internal/cmd/buildtool linux docker amd64

#help:
#help: The `make CLI/linux-static-armv6` command builds and statically links the
#help: ooniprobe and miniooni binaries for linux/arm/v6.
.PHONY: CLI/linux-static-armv6
CLI/linux-static-armv6:
	go run ./internal/cmd/buildtool linux docker armv6

#help:
#help: The `make CLI/linux-static-armv7` command builds and statically links the
#help: ooniprobe and miniooni binaries for linux/arm/v7.
.PHONY: CLI/linux-static-armv7
CLI/linux-static-armv7:
	go run ./internal/cmd/buildtool linux docker armv7

#help:
#help: The `make CLI/linux-static-arm64` command builds and statically links the
#help: ooniprobe and miniooni binaries for linux/arm64.
.PHONY: CLI/linux-static-arm64
CLI/linux-static-arm64:
	go run ./internal/cmd/buildtool linux docker arm64

#help:
#help: The `make CLI/miniooni` command creates a build of miniooni, for the current
#help: system, putting the binary in the top-level directory.
.PHONY: CLI/miniooni
CLI/miniooni:
	go run ./internal/cmd/buildtool generic miniooni

#help:
#help: The `make CLI/ooniprobe` command creates a build of ooniprobe, for the current
#help: system, putting the binary in the top-level directory.
.PHONY: CLI/ooniprobe
CLI/ooniprobe:
	go run ./internal/cmd/buildtool generic ooniprobe

#help:
#help: The `make CLI/windows` command builds the ooniprobe and miniooni
#help: command line clients for windows/386 and windows/amd64.
.PHONY: CLI/windows
CLI/windows:
	go run ./internal/cmd/buildtool windows

#help:
#help: The `make MOBILE/android` command builds the oonimkall library for Android.
.PHONY: MOBILE/android
MOBILE/android: search/for/go search/for/android/sdk maybe/copypsiphon
	./MOBILE/gomobile android ./pkg/oonimkall
	./MOBILE/android/createpom

#help:
#help: The `make MOBILE/ios` command builds the oonimkall library for iOS.
.PHONY: MOBILE/ios
MOBILE/ios: search/for/go search/for/zip search/for/xcode maybe/copypsiphon
	./MOBILE/gomobile ios ./pkg/oonimkall
	./MOBILE/ios/zipframework
	./MOBILE/ios/createpodspec

.PHONY: search/for/git
search/for/git:
	@printf "checking for git... "
	@command -v git || { echo "not found"; exit 1; }

.PHONY: search/for/go
search/for/go:
	./CLI/check-go-version

.PHONY: search/for/java
search/for/java:
	@printf "checking for java... "
	@command -v java || { echo "not found"; exit 1; }

.PHONY: search/for/xcode
search/for/xcode:
	./MOBILE/ios/check-xcode-version

.PHONY: search/for/zip
search/for/zip:
	@printf "checking for zip... "
	@command -v zip || { echo "not found"; exit 1; }

#
# Note: we check for files being already there before attempting
# to clone _because_ we put files in there using secrets when
# running cloud builds. This saves us from including a token with
# `repo` scope as a build secret, which is a very broad scope.
#
# Cloning the private repository, instead, is the way in which
# local builds get access to the psiphon config files.
#
.PHONY: maybe/copypsiphon
maybe/copypsiphon: search/for/git
	@if test "$(OONI_PSIPHON_TAGS)" = "ooni_psiphon_config"; then \
		if test ! -f ./internal/engine/psiphon-config.json.age -a \
		        ! -f ./internal/engine/psiphon-config.key; then \
			./script/copy-psiphon-files.bash $(GIT_CLONE_DIR) || exit 1; \
		fi; \
	fi

.PHONY: search/for/android/sdk
search/for/android/sdk: search/for/java
	./MOBILE/android/ensure

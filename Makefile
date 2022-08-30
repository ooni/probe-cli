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
#help: * GIT_CLONE_DIR         : directory where to clone repositories, by default
#help:                           set to `$HOME/.ooniprobe-build/src`.
GIT_CLONE_DIR = $(HOME)/.ooniprobe-build/src

#help:
#help: * OONI_PSIPHON_TAGS     : build tags for `go build -tags ...` that cause
#help:                           the build to embed a psiphon configuration file
#help:                           into the generated binaries. This build tag
#help:                           implies cloning the git@github.com:ooni/probe-private
#help:                           repository. If you do not have the permission to
#help:                           clone it, just clear this variable, e.g.:
#help:
#help:                               make OONI_PSIPHON_TAGS="" ./CLI/miniooni
OONI_PSIPHON_TAGS = ooni_psiphon_config

#quickhelp:
#quickhelp: The `make show-config` command shows the current value of the
#quickhelp: variables controlling the build.
.PHONY: show-config
show-config:
	@echo "GIT_CLONE_DIR=$(GIT_CLONE_DIR)"
	@echo "OONI_PSIPHON_TAGS=$(OONI_PSIPHON_TAGS)"

#help:
#help: The `make ./CLI/android` command builds miniooni and ooniprobe for
#help: all the supported Android architectures.
.PHONY: ./CLI/android
./CLI/android: search/for/go search/for/android/sdk maybe/copypsiphon
	./CLI/go-build-android 386 ./internal/cmd/miniooni
	./CLI/go-build-android 386 ./cmd/ooniprobe
	./CLI/go-build-android amd64 ./internal/cmd/miniooni
	./CLI/go-build-android amd64 ./cmd/ooniprobe
	./CLI/go-build-android arm ./internal/cmd/miniooni
	./CLI/go-build-android arm ./cmd/ooniprobe
	./CLI/go-build-android arm64 ./internal/cmd/miniooni
	./CLI/go-build-android arm64 ./cmd/ooniprobe

#help:
#help: The `make ./CLI/miniooni` command builds the miniooni experimental
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

#help:
#help: * `make ./CLI/miniooni-darwin-amd64`: darwin/amd64
.PHONY:   ./CLI/miniooni-darwin-amd64
./CLI/miniooni-darwin-amd64: search/for/go maybe/copypsiphon
	./CLI/go-build-cross darwin amd64 ./internal/cmd/miniooni

#help:
#help: * `make ./CLI/miniooni-darwin-arm64`: darwin/arm64
.PHONY:   ./CLI/miniooni-darwin-arm64
./CLI/miniooni-darwin-arm64: search/for/go maybe/copypsiphon
	./CLI/go-build-cross darwin arm64 ./internal/cmd/miniooni

#help:
#help: * `make ./CLI/miniooni-linux-386`: linux/386
.PHONY:   ./CLI/miniooni-linux-386
./CLI/miniooni-linux-386: search/for/go maybe/copypsiphon
	./CLI/go-build-cross linux 386 ./internal/cmd/miniooni

#help:
#help: * `make ./CLI/miniooni-linux-amd64`: linux/amd64
.PHONY:   ./CLI/miniooni-linux-amd64
./CLI/miniooni-linux-amd64: search/for/go maybe/copypsiphon
	./CLI/go-build-cross linux amd64 ./internal/cmd/miniooni

#help:
#help: * `make ./CLI/miniooni-linux-armv6`: linux/armv6
.PHONY:   ./CLI/miniooni-linux-armv6
./CLI/miniooni-linux-armv6: search/for/go maybe/copypsiphon
	./CLI/go-build-cross linux armv6 ./internal/cmd/miniooni

#help:
#help: * `make ./CLI/miniooni-linux-armv7`: linux/armv7
.PHONY:   ./CLI/miniooni-linux-armv7
./CLI/miniooni-linux-armv7: search/for/go maybe/copypsiphon
	./CLI/go-build-cross linux armv7 ./internal/cmd/miniooni

#help:
#help: * `make ./CLI/miniooni-linux-arm64`: linux/arm64
.PHONY:   ./CLI/miniooni-linux-arm64
./CLI/miniooni-linux-arm64: search/for/go maybe/copypsiphon
	./CLI/go-build-cross linux arm64 ./internal/cmd/miniooni

#help:
#help: * `make ./CLI/miniooni-windows-386.exe`: windows/386
.PHONY:   ./CLI/miniooni-windows-386.exe
./CLI/miniooni-windows-386.exe: search/for/go maybe/copypsiphon
	./CLI/go-build-cross windows 386 ./internal/cmd/miniooni

#help:
#help: * `make ./CLI/miniooni-windows-amd64.exe`: windows/amd64
.PHONY:   ./CLI/miniooni-windows-amd64.exe
./CLI/miniooni-windows-amd64.exe: search/for/go maybe/copypsiphon
	./CLI/go-build-cross windows amd64 ./internal/cmd/miniooni

#help:
#help: The `make ./CLI/ooniprobe-darwin` command builds the ooniprobe official
#help: command line client for darwin/amd64 and darwin/arm64.
#help:
#help: You can also build the following subtargets:
.PHONY: ./CLI/ooniprobe-darwin
./CLI/ooniprobe-darwin: ./CLI/ooniprobe-darwin-amd64 ./CLI/ooniprobe-darwin-arm64

#help:
#help: * `make ./CLI/ooniprobe-darwin-amd64`: darwin/amd64
.PHONY:     ./CLI/ooniprobe-darwin-amd64
./CLI/ooniprobe-darwin-amd64: search/for/go maybe/copypsiphon
	./CLI/go-build-darwin amd64 ./cmd/ooniprobe

#help:
#help: * `make ./CLI/ooniprobe-darwin-arm64`: darwin/arm64
.PHONY:     ./CLI/ooniprobe-darwin-arm64
./CLI/ooniprobe-darwin-arm64: search/for/go maybe/copypsiphon
	./CLI/go-build-darwin arm64 ./cmd/ooniprobe

#help:
#help: The `make ./CLI/ooniprobe-linux` command builds the ooniprobe official command
#help: line client for amd64, arm64, etc.
#help:
#help: You can also build the following subtargets:
.PHONY: ./CLI/ooniprobe-linux
./CLI/ooniprobe-linux: \
	./CLI/ooniprobe-linux-386 \
	./CLI/ooniprobe-linux-amd64 \
	./CLI/ooniprobe-linux-armv6 \
	./CLI/ooniprobe-linux-armv7 \
	./CLI/ooniprobe-linux-arm64

#help:
#help: * `make ./CLI/ooniprobe-linux-386`: linux/386
.PHONY:     ./CLI/ooniprobe-linux-386
./CLI/ooniprobe-linux-386: search/for/docker maybe/copypsiphon
	./CLI/go-build-linux-static 386 ./cmd/ooniprobe

#help:
#help: * `make ./CLI/ooniprobe-linux-amd64`: linux/amd64
.PHONY:     ./CLI/ooniprobe-linux-amd64
./CLI/ooniprobe-linux-amd64: search/for/docker maybe/copypsiphon
	./CLI/go-build-linux-static amd64 ./cmd/ooniprobe

#help:
#help: * `make ./CLI/ooniprobe-linux-armv6`: linux/arm
.PHONY:     ./CLI/ooniprobe-linux-armv6
./CLI/ooniprobe-linux-armv6: search/for/docker maybe/copypsiphon
	./CLI/go-build-linux-static armv6 ./cmd/ooniprobe

#help:
#help: * `make ./CLI/ooniprobe-linux-armv7`: linux/arm
.PHONY:     ./CLI/ooniprobe-linux-armv7
./CLI/ooniprobe-linux-armv7: search/for/docker maybe/copypsiphon
	./CLI/go-build-linux-static armv7 ./cmd/ooniprobe

#help:
#help: * `make ./CLI/ooniprobe-linux-arm64`: linux/arm64
.PHONY:     ./CLI/ooniprobe-linux-arm64
./CLI/ooniprobe-linux-arm64: search/for/docker maybe/copypsiphon
	./CLI/go-build-linux-static arm64 ./cmd/ooniprobe

#help:
#help: The `make ./CLI/ooniprobe-windows` command builds the ooniprobe official
#help: command line client for windows/386 and windows/amd64.
#help:
#help: You can also build the following subtargets:
.PHONY: ./CLI/ooniprobe-windows
./CLI/ooniprobe-windows: \
	./CLI/ooniprobe-windows-386.exe \
	./CLI/ooniprobe-windows-amd64.exe

#help:
#help: * `make ./CLI/ooniprobe-windows-386.exe`: windows/386
.PHONY:     ./CLI/ooniprobe-windows-386.exe
./CLI/ooniprobe-windows-386.exe: search/for/go search/for/mingw-w64 maybe/copypsiphon
	./CLI/go-build-windows 386 ./cmd/ooniprobe

#help:
#help: * `make ./CLI/ooniprobe-windows-amd64.exe`: windows/amd64
.PHONY:     ./CLI/ooniprobe-windows-amd64.exe
./CLI/ooniprobe-windows-amd64.exe: search/for/go search/for/mingw-w64 maybe/copypsiphon
	./CLI/go-build-windows amd64 ./cmd/ooniprobe

#help:
#help: The `make ./MOBILE/android` command builds the oonimkall library for Android.
#help:
#help: You can also build the following subtargets:
.PHONY: ./MOBILE/android
./MOBILE/android: ./MOBILE/android/oonimkall.aar ./MOBILE/android/oonimkall.pom

#help:
#help: * `make ./MOBILE/android/oonimkall.pom`: the POM
.PHONY:   ./MOBILE/android/oonimkall.pom
./MOBILE/android/oonimkall.pom:
	./MOBILE/android/createpom

#help:
#help: * `make ./MOBILE/android/oonimkall.aar`: the AAR
.PHONY:   ./MOBILE/android/oonimkall.aar
./MOBILE/android/oonimkall.aar: search/for/go search/for/android/sdk maybe/copypsiphon
	./MOBILE/gomobile android ./pkg/oonimkall

#help:
#help: The `make ./MOBILE/ios` command builds the oonimkall library for iOS.
#help:
#help: You can also build the following subtargets:
.PHONY: ./MOBILE/ios
./MOBILE/ios: ./MOBILE/ios/oonimkall.xcframework.zip ./MOBILE/ios/oonimkall.podspec

#help:
#help: * `make ./MOBILE/ios/oonimkall.xcframework.zip`: zip the xcframework
.PHONY:   ./MOBILE/ios/oonimkall.xcframework.zip
./MOBILE/ios/oonimkall.xcframework.zip: search/for/zip ./MOBILE/ios/oonimkall.xcframework
	./MOBILE/ios/zipframework

#help:
#help: * `make ./MOBILE/ios/xcframework`: the xcframework
.PHONY:     ./MOBILE/ios/oonimkall.xcframework
./MOBILE/ios/oonimkall.xcframework: search/for/go search/for/xcode maybe/copypsiphon
	./MOBILE/gomobile ios ./pkg/oonimkall

#help:
#help: * `make ./MOBILE/ios/oonimkall.podspec`: the podspec
.PHONY:   ./MOBILE/ios/oonimkall.podspec
./MOBILE/ios/oonimkall.podspec: ./MOBILE/ios/template.podspec
	./MOBILE/ios/createpodspec

.PHONY: search/for/docker
search/for/docker:
	@printf "checking for docker... "
	@command -v docker || { echo "not found"; exit 1; }

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

.PHONY: search/for/mingw-w64
search/for/mingw-w64:
	./CLI/check-mingw-w64-version

.PHONY: search/for/xcode
search/for/xcode:
	./MOBILE/ios/check-xcode-version

.PHONY: search/for/zip
search/for/zip:
	@printf "checking for zip... "
	@command -v zip || { echo "not found"; exit 1; }

#help:
#help: The `make maybe/copypsiphon` command checks whether we want
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

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
#help: The `make CLI/darwin` command builds the ooniprobe and miniooni
#help: command line clients for darwin/amd64 and darwin/arm64.
.PHONY: CLI/darwin
CLI/darwin:
	./script/go.bash run ./internal/cmd/buildtool darwin

#help:
#help: The `make CLI/linux-static-386` command builds and statically links the
#help: ooniprobe and miniooni binaries for linux/386.
.PHONY: CLI/linux-static-386
CLI/linux-static-386:
	./script/go.bash run ./internal/cmd/buildtool linux docker 386

#help:
#help: The `make CLI/linux-static-amd64` command builds and statically links the
#help: ooniprobe and miniooni binaries for linux/amd64.
.PHONY: CLI/linux-static-amd64
CLI/linux-static-amd64:
	./script/go.bash run ./internal/cmd/buildtool linux docker amd64

#help:
#help: The `make CLI/linux-static-armv6` command builds and statically links the
#help: ooniprobe and miniooni binaries for linux/arm/v6.
.PHONY: CLI/linux-static-armv6
CLI/linux-static-armv6:
	./script/go.bash run ./internal/cmd/buildtool linux docker armv6

#help:
#help: The `make CLI/linux-static-armv7` command builds and statically links the
#help: ooniprobe and miniooni binaries for linux/arm/v7.
.PHONY: CLI/linux-static-armv7
CLI/linux-static-armv7:
	./script/go.bash run ./internal/cmd/buildtool linux docker armv7

#help:
#help: The `make CLI/linux-static-arm64` command builds and statically links the
#help: ooniprobe and miniooni binaries for linux/arm64.
.PHONY: CLI/linux-static-arm64
CLI/linux-static-arm64:
	./script/go.bash run ./internal/cmd/buildtool linux docker arm64

#help:
#help: The `make CLI/miniooni` command creates a build of miniooni, for the current
#help: system, putting the binary in the top-level directory.
.PHONY: CLI/miniooni
CLI/miniooni:
	./script/go.bash run ./internal/cmd/buildtool generic miniooni

#help:
#help: The `make CLI/ooniprobe` command creates a build of ooniprobe, for the current
#help: system, putting the binary in the top-level directory.
.PHONY: CLI/ooniprobe
CLI/ooniprobe:
	./script/go.bash run ./internal/cmd/buildtool generic ooniprobe

#help:
#help: The `make CLI/windows` command builds the ooniprobe and miniooni
#help: command line clients for windows/386 and windows/amd64.
.PHONY: CLI/windows
CLI/windows:
	./script/go.bash run ./internal/cmd/buildtool windows

#help:
#help: The `make android` command builds the oonimkall library for Android
#help: and compiles miniooni and ooniprobe for android CLI usage.
.PHONY: android
android: search/for/java
	./script/go.bash run ./internal/cmd/buildtool android cdeps zlib openssl libevent tor
	./script/go.bash run ./internal/cmd/buildtool android cli
	./script/go.bash run ./internal/cmd/buildtool android gomobile

#help:
#help: The `make ios` command builds the oonimkall library for iOS.
.PHONY: ios
ios: search/for/zip search/for/xcode
	./script/go.bash run ./internal/cmd/buildtool ios cdeps zlib openssl libevent tor
	./script/go.bash run ./internal/cmd/buildtool ios gomobile
	./MOBILE/ios/make-extra-frameworks
	./MOBILE/ios/zipframeworks
	./MOBILE/ios/createpodspecs

#help:
#help: The `make DESKTOP/windows` command builds the oonimkall jar for windows.
.PHONY: DESKTOP/windows
DESKTOP/windows: search/for/java
	go run ./internal/cmd/buildtool desktop oomobile --target=windows

#help:
#help: The `make DESKTOP/darwin` command builds the oonimkall jar for darwin.
.PHONY: DESKTOP/darwin
DESKTOP/darwin: search/for/java
	./script/go.bash run ./internal/cmd/buildtool desktop oomobile --target=darwin

#help:
#help: The `make DESKTOP/linux` command builds the oonimkall jar for linux.
.PHONY: DESKTOP/linux
DESKTOP/linux: search/for/java
	./script/go.bash run ./internal/cmd/buildtool desktop oomobile --target=linux

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

#help:
#help: The `make docs clean` command builds the docs for docs.ooni.org.
.PHONY: docs clean
docs:
	./script/build_docs.sh

clean:
	rm -rf dist/

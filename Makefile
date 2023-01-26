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
#help: The `make CLI/android` command builds miniooni and ooniprobe for android.
.PHONY: CLI/android
CLI/android:
	go run ./internal/cmd/buildtool android cli

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
MOBILE/android: search/for/java
	go run ./internal/cmd/buildtool android gomobile
	./MOBILE/android/createpom

#help:
#help: The `make MOBILE/ios` command builds the oonimkall library for iOS.
.PHONY: MOBILE/ios
MOBILE/ios: search/for/zip search/for/xcode
	go run ./internal/cmd/buildtool ios gomobile
	./MOBILE/ios/zipframework
	./MOBILE/ios/createpodspec

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

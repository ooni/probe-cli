__GOVERSION = 1.16.3
XCODEVERSION = 12.5

GOVERSION = go$(__GOVERSION)
GODOCKER = golang:$(__GOVERSION)-alpine

#quickhelp: Usage: make [VARIABLE=VALUE ...] TARGET ...
.PHONY: quickhelp
quickhelp:
	@cat Makefile | grep '^#quickhelp:' | sed -e 's/^#quickhelp://' -e 's/^\ *//'

#quickhelp:
#quickhelp: The `make printtargets` command prints all available targets.
.PHONY: printtargets
printtargets:
	@cat Makefile | grep '^\.PHONY:' | sed -e 's/^\.PHONY://' -e 's/^/*/'

#quickhelp:
#quickhelp: The `make help' command provides detailed usage instructions. We
#quickhelp: recommend running `make help|less' to page the output.
.PHONY: help
help:
	@cat Makefile | grep -E '^#(quick)?help:' | sed -E -e 's/^#(quick)?help://' -e s'/^\ //'

#help:
#help: The following variables control the build. You can specify them
#help: before the targets as indicated above in the usage line.
#help:
#help: * GITCLONEDIR         : directory where to clone repositories, by default
#help:                         set to `$HOME/.ooniprobe-build/src'.
GITCLONEDIR = $(HOME)/.ooniprobe-build/src

$(GITCLONEDIR):
	mkdir -p $(GITCLONEDIR)

#help:
#help: * GOEXTRAFLAGS        : extra flags passed to `go build ...`, empty by
#help:                         default. Useful to pass flags to `go`, e.g.:
#help:
#help:                             make GOEXTRAFLAGS="-x -v" miniooni
GOEXTRAFLAGS =

#help:
#help: * GPGUSER             : allows overriding the default GPG user used
#help:                         to sign binary releases, e.g.:
#help:
#help:                             make GPGPUSER=john@doe.com ooniprobe/windows
GPGUSER = simone@openobservatory.org

#help:
#help: * OONIPSIPHONTAGS     : build tags for `go build -tags ...` that cause
#help:                         the build to embed a psiphon configuration file
#help:                         into the generated binaries. This build tag
#help:                         implies cloning the git@github.com:ooni/probe-private
#help:                         repository. If you do not have the permission to
#help:                         clone ooni-private just clear this variable, e.g.:
#help:
#help:                             make OONIPSIPHONTAGS="" miniooni
OONIPSIPHONTAGS = ooni_psiphon_config

#help:
#help: The `make printvars` command prints the current value of the above
#help: listed build-controlling variables.
.PHONY: printvars
printvars:
	@echo "GITCLONEDIR=$(GITCLONEDIR)"
	@echo "GOEXTRAFLAGS=$(GOEXTRAFLAGS)"
	@echo "OONIPSIPHONTAGS=$(OONIPSIPHONTAGS)"

#help:
#help: The `make miniooni' command builds the miniooni experimental
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
#help: * `make ./CLI/darwin/amd64/miniooni': darwin/amd64
.PHONY: ./CLI/darwin/amd64/miniooni
./CLI/darwin/amd64/miniooni: configure/go maybe/copypsiphon
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -tags="$(OONIPSIPHONTAGS)" -ldflags="-s -w" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `make ./CLI/darwin/arm64/miniooni': darwin/arm64
.PHONY: ./CLI/darwin/arm64/miniooni
./CLI/darwin/arm64/miniooni: configure/go maybe/copypsiphon
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -tags="$(OONIPSIPHONTAGS)" -ldflags="-s -w" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `make ./CLI/linux/386/miniooni': linux/386
.PHONY: ./CLI/linux/386/miniooni
./CLI/linux/386/miniooni: configure/go maybe/copypsiphon
	GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -tags="netgo,$(OONIPSIPHONTAGS)" -ldflags="-s -w -extldflags -static" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `make ./CLI/linux/amd64/miniooni': linux/amd64
.PHONY: ./CLI/linux/amd64/miniooni
./CLI/linux/amd64/miniooni: configure/go maybe/copypsiphon
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags="netgo,$(OONIPSIPHONTAGS)" -ldflags="-s -w -extldflags -static" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * `make ./CLI/linux/arm/miniooni': linux/arm
.PHONY: ./CLI/linux/arm/miniooni
./CLI/linux/arm/miniooni: configure/go maybe/copypsiphon
	GOOS=linux GOARCH=arm CGO_ENABLED=0 GOARM=7 go build -tags="netgo,$(OONIPSIPHONTAGS)" -ldflags="-s -w -extldflags -static" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * make `./CLI/linux/arm64/miniooni': linux/arm64
.PHONY: ./CLI/linux/arm64/miniooni
./CLI/linux/arm64/miniooni: configure/go maybe/copypsiphon
	GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags="netgo,$(OONIPSIPHONTAGS)" -ldflags="-s -w -extldflags -static" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * make `./CLI/windows/386/miniooni.exe': windows/386
.PHONY: ./CLI/windows/386/miniooni.exe
./CLI/windows/386/miniooni.exe: configure/go maybe/copypsiphon
	GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -tags="$(OONIPSIPHONTAGS)" -ldflags="-s -w" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: * make `./CLI/windows/amd64/miniooni.exe': windows/amd64
.PHONY: ./CLI/windows/amd64/miniooni.exe
./CLI/windows/amd64/miniooni.exe: configure/go maybe/copypsiphon
	GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags="$(OONIPSIPHONTAGS)" -ldflags="-s -w" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

#help:
#help: The `make ooniprobe/darwin' command builds the ooniprobe official
#help: command line client for darwin/amd64 and darwin/arm64.
#help:
#help: We also support the following commands:
.PHONY: ooniprobe/darwin
ooniprobe/darwin:                    \
	./CLI/darwin/amd64/ooniprobe.asc \
	./CLI/darwin/arm64/ooniprobe.asc

.PHONY: ./CLI/darwin/amd64/ooniprobe.asc
./CLI/darwin/amd64/ooniprobe.asc: ./CLI/darwin/amd64/ooniprobe
	rm -f $@ && gpg -abu $(GPGUSER) $<

#help:
#help: * `make ./CLI/darwin/amd64/ooniprobe': darwin/amd64
.PHONY: ./CLI/darwin/amd64/ooniprobe
./CLI/darwin/amd64/ooniprobe: configure/go maybe/copypsiphon
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=1 go build -tags="$(OONIPSIPHONTAGS)" -ldflags="-s -w" $(GOEXTRAFLAGS) -o $@ ./cmd/ooniprobe

.PHONY: ./CLI/darwin/arm64/ooniprobe.asc
./CLI/darwin/arm64/ooniprobe.asc: ./CLI/darwin/arm64/ooniprobe
	rm -f $@ && gpg -abu $(GPGUSER) $<

#help:
#help: * `make ./CLI/darwin/arm64/ooniprobe': darwin/arm64
.PHONY: ./CLI/darwin/arm64/ooniprobe
./CLI/darwin/arm64/ooniprobe: configure/go maybe/copypsiphon
	GOOS=darwin GOARCH=arm64 CGO_ENABLED=1 go build -tags="$(OONIPSIPHONTAGS)" -ldflags="-s -w" $(GOEXTRAFLAGS) -o $@ ./cmd/ooniprobe

#help:
#help: The `make ooniprobe/debian' command builds the ooniprobe CLI
#help: debian package for amd64 and arm64.
#help:
#help: We also support the following commands:
.PHONY: ooniprobe/debian
ooniprobe/debian:           \
	ooniprobe/debian/amd64  \
	ooniprobe/debian/arm64

#help:
#help: * `make ooniprobe/debian/amd64': debian/amd64
.PHONY: ooniprobe/debian/amd64
ooniprobe/debian/amd64: configure/docker ./CLI/linux/amd64/ooniprobe
	docker pull --platform linux/amd64 debian:stable
	docker run --platform linux/amd64 -v `pwd`:/ooni -w /ooni debian:stable ./CLI/linux/debian

#help:
#help: * `make ooniprobe/debian/arm64': debian/arm64
.PHONY: ooniprobe/debian/arm64
ooniprobe/debian/arm64: configure/docker ./CLI/linux/arm64/ooniprobe
	docker pull --platform linux/arm64 debian:stable
	docker run --platform linux/arm64 -v `pwd`:/ooni -w /ooni debian:stable ./CLI/linux/debian

#help:
#help: The `make ooniprobe/linux' command builds the ooniprobe official command
#help: line client for amd64 and arm64.
#help:
#help: We also support the following commands:
.PHONY: ooniprobe/linux
ooniprobe/linux:                     \
	./CLI/linux/amd64/ooniprobe.asc  \
	./CLI/linux/arm64/ooniprobe.asc

.PHONY: ./CLI/linux/amd64/ooniprobe.asc
./CLI/linux/amd64/ooniprobe.asc: ./CLI/linux/amd64/ooniprobe
	rm -f $@ && gpg -abu $(GPGUSER) $<

#help:
#help: * `make ./CLI/linux/amd64/ooniprobe': linux/amd64
.PHONY: ./CLI/linux/amd64/ooniprobe
./CLI/linux/amd64/ooniprobe: configure/docker maybe/copypsiphon
	docker pull --platform linux/amd64 $(GODOCKER)
	docker run --platform linux/amd64 -e GOARCH=amd64 -v `pwd`:/ooni -w /ooni $(GODOCKER) ./CLI/linux/build -tags=netgo,$(OONIPSIPHONTAGS)

.PHONY: ./CLI/linux/arm64/ooniprobe.asc
./CLI/linux/arm64/ooniprobe.asc: ./CLI/linux/arm64/ooniprobe
	rm -f $@ && gpg -abu $(GPGUSER) $<

#help:
#help: * `make ./CLI/linux/arm64/ooniprobe': linux/arm64
.PHONY: ./CLI/linux/arm64/ooniprobe
./CLI/linux/arm64/ooniprobe: configure/docker maybe/copypsiphon
	docker pull --platform linux/arm64 $(GODOCKER)
	docker run --platform linux/arm64 -e GOARCH=arm64 -v `pwd`:/ooni -w /ooni $(GODOCKER) ./CLI/linux/build -tags=netgo,$(OONIPSIPHONTAGS)

#help:
#help: The `make ooniprobe/windows' command builds the ooniprobe official
#help: command line client for windows/386 and windows/amd64.
#help:
#help: We also support the following commands:
.PHONY: ooniprobe/windows
ooniprobe/windows:                         \
	./CLI/windows/386/ooniprobe.exe.asc    \
	./CLI/windows/amd64/ooniprobe.exe.asc

.PHONY: ./CLI/windows/386/ooniprobe.exe.asc
./CLI/windows/386/ooniprobe.exe.asc: ./CLI/windows/386/ooniprobe.exe
	rm -f $@ && gpg -abu $(GPGUSER) $<

#help:
#help: * `make ./CLI/windows/386/ooniprobe.exe': windows/386
.PHONY: ./CLI/windows/386/ooniprobe.exe
./CLI/windows/386/ooniprobe.exe: configure/go configure/mingw-w64 maybe/copypsiphon
	GOOS=windows GOARCH=386 CGO_ENABLED=1 CC=i686-w64-mingw32-gcc go build -tags="$(OONIPSIPHONTAGS)" -ldflags="-s -w" $(GOEXTRAFLAGS) -o $@ ./cmd/ooniprobe

.PHONY: ./CLI/windows/amd64/ooniprobe.exe.asc
./CLI/windows/amd64/ooniprobe.exe.asc: ./CLI/windows/amd64/ooniprobe.exe
	rm -f $@ && gpg -abu $(GPGUSER) $<

#help:
#help: * `make ./CLI/windows/amd64/ooniprobe.exe': windows/amd64
.PHONY: ./CLI/windows/amd64/ooniprobe.exe
./CLI/windows/amd64/ooniprobe.exe: configure/go configure/mingw-w64 maybe/copypsiphon
	GOOS=windows GOARCH=amd64 CGO_ENABLED=1 CC=x86_64-w64-mingw32-gcc go build -tags="$(OONIPSIPHONTAGS)" -ldflags="-s -w" $(GOEXTRAFLAGS) -o $@ ./cmd/ooniprobe

#help:
#help: The `make ios` command builds the oonimkall library for iOS.
#help:
#help: We also support the following commands:
.PHONY: ios
ios:                                      \
	./MOBILE/ios/oonimkall.framework.zip  \
	./MOBILE/ios/oonimkall.podspec

#help:
#help: * `make ./MOBILE/ios/oonimkall.framework.zip': zip the framework
.PHONY: ./MOBILE/ios/oonimkall.framework.zip
./MOBILE/ios/oonimkall.framework.zip: configure/zip ./MOBILE/ios/oonimkall.framework
	cd ./MOBILE/ios && rm -rf oonimkall.framework.zip
	cd ./MOBILE/ios && zip -yr oonimkall.framework.zip oonimkall.framework


#help:
#help: * `make ./MOBILE/ios/framework': the framework
.PHONY: ./MOBILE/ios/oonimkall.framework
./MOBILE/ios/oonimkall.framework: configure/go configure/xcode
	go get -u golang.org/x/mobile/cmd/gomobile@latest
	$(GOMOBILE) init
	$(GOMOBILE) bind -target ios -o $@ -tags="$(OONIPSIPHONTAGS)" -ldflags '-s -w' ./pkg/oonimkall

GOMOBILE = `go env GOPATH`/bin/gomobile

#help:
#help: * `make ./MOBILE/ios/oonimkall.podspec': the podspec
./MOBILE/ios/oonimkall.podspec: ./MOBILE/template.podspec
	cat $< | sed -e 's/@VERSION@/$(OONIMKALL_V)/g' -e 's/@RELEASE@/$(OONIMKALL_R)/g' > $@

OONIMKALL_V = `date -u +%Y.%m.%d-%H%M%S`
OONIMKALL_R = `git describe --tags`

#help:
#help: The `make configure/go` command ensures the `go` executable is
#help: in your `PATH` and we are using the expected version.
.PHONY: configure/go
configure/go:
	@printf "checking for go... "
	@command -v go || { echo "not found"; exit 1; }
	@printf "checking for go version... "
	@echo $(__GOVERSION_REAL)
	@[ "$(GOVERSION)" = "$(__GOVERSION_REAL)" ] || { echo "fatal: go version must be $(GOVERSION) instead of $(__GOVERSION_REAL)"; exit 1; }

# $(__GOVERSION_REAL) is the Go version according to the `go` executable.
__GOVERSION_REAL=$$(go version | awk '{print $$3}')

#help:
#help: The `make configure/docker` command ensures `docker` is available.
.PHONY: configure/docker
configure/docker:
	@printf "checking for docker... "
	@command -v git || { echo "not found"; exit 1; }

#help:
#help: The `make configure/git` command ensures `git` is available.
.PHONY: configure/git
configure/git:
	@printf "checking for git... "
	@command -v git || { echo "not found"; exit 1; }

#help:
#help: The `make configure/mingw-w64` command ensures `mingw-w64` is installed.
.PHONY: configure/mingw-w64
configure/mingw-w64:
	@printf "checking for x86_64-w64-mingw32-gcc... "
	@command -v x86_64-w64-mingw32-gcc || { echo "not found"; exit 1; }
	@printf "checking for i686-w64-mingw32-gcc... "
	@command -v i686-w64-mingw32-gcc || { echo "not found"; exit 1; }

#help:
#help: The `make configure/xcode` command ensures `Xcode` is available.
.PHONY: configure/xcode
configure/xcode:
	@printf "checking for xcodebuild... "
	@command -v xcodebuild || { echo "not found"; exit 1; }
	@printf "checking for Xcode version... "
	@echo $(__XCODEVERSION_REAL)
	@[ "$(XCODEVERSION)" = "$(__XCODEVERSION_REAL)" ] || { echo "fatal: Xcode version must be $(XCODEVERSION) instead of $(__XCODEVERSION_REAL)"; exit 1; }

__XCODEVERSION_REAL = `xcodebuild -version | grep ^Xcode | awk '{print $$2}'`

#help:
#help: The `make configure/zip` command ensures `zip` is available.
.PHONY: configure/zip
configure/zip:
	@printf "checking for zip... "
	@command -v zip || { echo "not found"; exit 1; }

#help:
#help: The `make maybe/copypsiphon' command copies private psiphon configuration file
#help: into the current tree unless `$(OONIPSIPHONTAGS)' is empty.
.PHONY: maybe/copypsiphon
maybe/copypsiphon: configure/git
	test -z "$(OONIPSIPHONTAGS)" || $(MAKE) -f Makefile $(OONIPRIVATE)
	test -z "$(OONIPSIPHONTAGS)" || cp $(OONIPRIVATE)/psiphon-config.key ./internal/engine
	test -z "$(OONIPSIPHONTAGS)" || cp $(OONIPRIVATE)/psiphon-config.json.age ./internal/engine

# OONIPRIVATE is the directory where we clone the private repository.
OONIPRIVATE = $(GITCLONEDIR)/github.com/ooni/probe-private

# OONIPRIVATE_REPO is the private repository URL.
OONIPRIVATE_REPO = git@github.com:ooni/probe-private

# $(OONIPRIVATE) clones the private repository in $(GITCLONEDIR)
$(OONIPRIVATE): $(GITCLONEDIR)
	git clone $(OONIPRIVATE_REPO) $(OONIPRIVATE)

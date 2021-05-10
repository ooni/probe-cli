#help:
#help: # OONI Probe CLI Makefile
#help:
#help: ```
#help: Usage: make [VARIABLE=VALUE ...] <target> ...
#help: ```
#help:
#help: This Makefile helps you to build the following pieces of software:
#help:
#help: * ooniprobe           : the official CLI client
#help: * oonimkall           : the mobile library
#help: * miniooni            : the research CLI client
#help:
#help: In the following, we describe all the top-level targets.
#help:

#help: # Quick help (target: `quickhelp`)
#help:
#help: By running `make` or `make quickhelp` you get a list of all
#help: the toplevel target supported by this Makefile.
#help:
.PHONY: quickhelp
quickhelp:
	@echo "Usage: make [VARIABLE=VALUE ...] target ..."
	@echo ""
	@echo "Available targets:"
	@echo ""
	@cat Makefile|grep '^\.PHONY'|sed -e 's/^\.PHONY:/*/g'
	@echo ""
	@echo "Try 'make help' for more help."

.PHONY: help
help:
	@cat Makefile|grep '^#help:'|sed -e 's/^#help://g' -e 's/^ //g'

#help: ## User-overridable variables (target: `printenv`)
#help:
#help: * GITCLONEDIR         : directory where we clone private repositories.
#help:
GITCLONEDIR = $(HOME)/.ooniprobe-build/src

# $(GITCLONEDIR) creates the directory
$(GITCLONEDIR):
	mkdir -p $(GITCLONEDIR)

#help: * GOEXTRAFLAGS        : extra flags passed to `go build ...`, empty by
#help:                         default. Use to pass `-v` or `-x` to `go`.
#help:
GOEXTRAFLAGS =

#help: * GOSDKHOME           : directory where to install the Go SDK. We download
#help:                         the Go SDK from golang.org/dl using `get get` to make
#help:                         sure we always use the SDK version we want.
#help:
GOSDKHOME = $(HOME)/sdk

# $(GOSDKHOME) creates the SDK directory
$(GOSDKHOME):
	mkdir -p $(GOSDKHOME)

#help: * OONIPSIPHONTAGS     : build tags for `go build -tags ...` that cause
#help:                         the build to embed a psiphon configuration file
#help:                         into the generated binaries. This build tag
#help:                         implies cloning the git@github.com:ooni/probe-private
#help:                         repository. If you do not have the permission to
#help:                         clone this repository, just clear this variable, e.g.:
#help:
#help:                             make OONIPSIPHONTAGS="" miniooni
#help:
OONIPSIPHONTAGS = ooni_psiphon_config,

#help: Use `make printenv` to print user-overridable variables.
#help:
.PHONY: printenv
printenv:
	@echo "GITCLONEDIR=$(GITCLONEDIR)"
	@echo "GOEXTRAFLAGS=$(GOEXTRAFLAGS)"
	@echo "GOSDKHOME=$(GOSDKHOME)"
	@echo "OONIPSIPHONTAGS=$(OONIPSIPHONTAGS)"

#help: ## Go SDK (target: `gosdk`)
#help:
#help: We download a specific version of the Go SDK. The `make gosdk`
#help: command ensures we download the SDK and prints information about
#help: what SDK we downloaded and where it is installed.
#help:
.PHONY: gosdk
gosdk: $(GOSDK)
	@echo "GOPATH=$(GOPATH)"
	@echo "GOSDK=$(GOSDK)"

# GOVERSION is the version of Go we use.
GOVERSION = 1.16.4

# GOPATH is the path where Go installs binaries.
GOPATH = $(HOME)/go

# GOSDK is the Go SDK we use.
GOSDK = $(GOSDKHOME)/go$(GOVERSION)

# $(GOSDK) ensures we have a Go SDK.
$(GOSDK): $(GOSDKHOME)
	go get $(GOEXTRAFLAGS) golang.org/dl/go$(GOVERSION)
	$(GOPATH)/bin/go$(GOVERSION) download

#help: ## Embedded psiphon config (taget: `maybe/copypsiphon`)
#help:
#help: When $(OONIPSIPHONTAGS) is not empty, we clone git@github.com:ooni/probe-private
#help: in $(GITCLONEDIR) and we copy private psiphon configuration into the current
#help: tree. In turn, we pass $(OONIPSIPHONTAGS) to `go build...` so that these config
#help: files are embedded directly into the builds.
#help:
.PHONY: maybe/copypsiphon
maybe/copypsiphon:
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

#help: ## Miniooni (target: `miniooni`)
#help:
#help: By running `make miniooni` you will build the miniooni research
#help: client for all available platforms and operating systems.
#help:
#help: We also define an individual target for every available platform
#help: and system (e.g., `make CLI/linux/amd64/miniooni`).
#help:

.PHONY: CLI/linux/amd64/miniooni
CLI/linux/amd64/miniooni: $(GOSDK) maybe/copypsiphon
	PATH=$(GOSDK)/bin:$(PATH) GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags "$(OONIPSIPHONTAGS)netgo" -ldflags="-s -w -extldflags -static" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

.PHONY: CLI/linux/386/miniooni
CLI/linux/386/miniooni: $(GOSDK) maybe/copypsiphon
	PATH=$(GOSDK)/bin:$(PATH) GOOS=linux GOARCH=386 CGO_ENABLED=0 go build -tags "$(OONIPSIPHONTAGS)netgo" -ldflags="-s -w -extldflags -static" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

.PHONY: CLI/linux/amd64/miniooni
CLI/linux/arm64/miniooni: $(GOSDK) maybe/copypsiphon
	PATH=$(GOSDK)/bin:$(PATH) GOOS=linux GOARCH=arm64 CGO_ENABLED=0 go build -tags "$(OONIPSIPHONTAGS)netgo" -ldflags="-s -w -extldflags -static" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

.PHONY: CLI/linux/arm/miniooni
CLI/linux/arm/miniooni: $(GOSDK) maybe/copypsiphon
	PATH=$(GOSDK)/bin:$(PATH) GOOS=linux GOARCH=arm CGO_ENABLED=0 GOARM=7 go build -tags "$(OONIPSIPHONTAGS)netgo" -ldflags="-s -w -extldflags -static" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

.PHONY: CLI/darwin/amd64/miniooni
CLI/darwin/amd64/miniooni: $(GOSDK) maybe/copypsiphon
	PATH=$(GOSDK)/bin:$(PATH) GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build -tags "$(OONIPSIPHONTAGS)" -ldflags="-s -w" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

.PHONY: CLI/darwin/arm64/miniooni
CLI/darwin/arm64/miniooni: $(GOSDK) maybe/copypsiphon
	PATH=$(GOSDK)/bin:$(PATH) GOOS=darwin GOARCH=arm64 CGO_ENABLED=0 go build -tags "$(OONIPSIPHONTAGS)" -ldflags="-s -w" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

.PHONY: CLI/windows/386/miniooni.exe
CLI/windows/386/miniooni.exe: $(GOSDK) maybe/copypsiphon
	PATH=$(GOSDK)/bin:$(PATH) GOOS=windows GOARCH=386 CGO_ENABLED=0 go build -tags "$(OONIPSIPHONTAGS)" -ldflags="-s -w" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

.PHONY: CLI/windows/amd64/miniooni.exe
CLI/windows/amd64/miniooni.exe: $(GOSDK) maybe/copypsiphon
	PATH=$(GOSDK)/bin:$(PATH) GOOS=windows GOARCH=amd64 CGO_ENABLED=0 go build -tags "$(OONIPSIPHONTAGS)" -ldflags="-s -w" $(GOEXTRAFLAGS) -o $@ ./internal/cmd/miniooni

MINIOONI_TARGETS =                     \
		CLI/linux/amd64/miniooni       \
		CLI/linux/386/miniooni         \
		CLI/linux/arm64/miniooni       \
		CLI/linux/arm/miniooni         \
		CLI/darwin/amd64/miniooni      \
		CLI/darwin/arm64/miniooni      \
		CLI/windows/386/miniooni.exe   \
		CLI/windows/amd64/miniooni.exe

.PHONY: miniooni
miniooni: $(MINIOONI_TARGETS)

#help: ## ooniprobe (targets: `ooniprobe/windows`, `ooniprobe/linux`, `ooniprobe/darwin`)
#help:
#help: We define targets for building ooniprobe for windows, linux, and darwin (i.e.,
#help: macOS). Building for linux requires docker. Building for windows requires a
#help: working mingw-w64 installation. Building for macOS/darwin requires the Xcode
#help: command line tools (i.e., you must be on macOS).
#help:
#help: We also define individual targets (e.g., `CLI/linux/arm64/ooniprobe`).
#help:

.PHONY: CLI/windows/amd64/ooniprobe.exe
CLI/windows/amd64/ooniprobe.exe: $(GOSDK) maybe/copypsiphon
	command -v x86_64-w64-mingw32-gcc
	PATH=$(GOSDK)/bin:$(PATH) GOOS=windows GOARCH=amd64 CC=x86_64-w64-mingw32-gcc go build -tags "$(OONIPSIPHONTAGS)" -ldflags="-s -w" $(GOEXTRAFLAGS) -o $@ ./cmd/ooniprobe

.PHONY: CLI/windows/386/ooniprobe.exe
CLI/windows/386/ooniprobe.exe: $(GOSDK) maybe/copypsiphon
	command -v i686-w64-mingw32-gcc
	PATH=$(GOSDK)/bin:$(PATH) GOOS=windows GOARCH=386 CC=i686-w64-mingw32-gcc go build -tags "$(OONIPSIPHONTAGS)" -ldflags="-s -w" $(GOEXTRAFLAGS) -o $@ ./cmd/ooniprobe

.PHONY: ooniprobe/windows
ooniprobe/windows: CLI/windows/amd64/ooniprobe.exe CLI/windows/386/ooniprobe.exe

.PHONY: ooniprobe/linux
ooniprobe/linux:

.PHONY: ooniprobe/darwin
ooniprobe/darwin:

#help: ## OONI Probe Android Library (target: `oonimkall/android`)
#help:
#help: This target builds oonimkall (i.e., OONI Probe's mobile library) for Android.
#help:

.PHONY: oonimkall/android
oonimkall/android:

#help: ## OONI Probe iOS Library (target: `oonimkall/ios`)
#help:
#help: This target builds oonimkall (i.e., OONI Probe's mobile library) for iOS.
#help:

.PHONY: oonimkall/ios
oonimkall/ios:

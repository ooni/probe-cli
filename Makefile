GO ?= go

install-dev-deps:
	@$(GO) get golang.org/x/tools/cmd/cover
	@$(GO) get github.com/mattn/goveralls

build:
	@echo "Building dist/ooni"
	@$(GO) build -o dist/ooni cmd/ooni/main.go
.PHONY: build

build-windows:
	@echo "Building dist/windows/amd64/ooni.exe"
	@./build.sh windows

build-linux:
	@echo "Building dist/linux/amd64/ooni"
	@./build.sh linux

build-macos:
	@echo "Building dist/macos/amd64/ooni"
	@./build.sh macos

build-all: build-windows build-linux build-macos
.PHONY: build-all build-windows build-linux build-macos

bindata:
	@$(GO) run vendor/github.com/shuLhan/go-bindata/go-bindata/*.go \
		-nometadata	\
		-o internal/bindata/bindata.go -pkg bindata \
	    data/...;

release:
	goreleaser release

test-internal:
	@$(GO) test -v ./internal/...

.PHONY: bindata

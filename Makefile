GO ?= go

install-dev-deps:
	@$(GO) get -u github.com/golang/dep/...
	@$(GO) get golang.org/x/tools/cmd/cover
	@$(GO) get github.com/mattn/goveralls

build:
	@echo "Building dist/ooni"
	@$(GO) build -i -o dist/ooni cmd/ooni/main.go
.PHONY: build

build-windows:
	@echo "Building dist/ooni.exe"
	CC=x86_64-w64-mingw32-gcc CXX=x86_64-w64-mingw32-g++ GOOS=windows GOARCH=amd64 CGO_ENABLED=1 go build -o dist/ooni.exe -x cmd/ooni/main.go

build-all: build build-windows
.PHONY: build-all

bindata:
	@$(GO) run vendor/github.com/shuLhan/go-bindata/go-bindata/*.go \
		-nometadata	\
		-o internal/bindata/bindata.go -pkg bindata \
	    data/...;

test-internal:
	@$(GO) test -v ./internal/...

.PHONY: bindata
